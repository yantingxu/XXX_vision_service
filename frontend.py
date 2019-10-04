#coding=utf-8
import os
import time
import datetime
import json
import logging
import itertools
#from urllib import urlencode
from urllib.parse import urlencode
import tornado.ioloop
import tornado.web
from tornado import gen
from tornado.httpclient import AsyncHTTPClient, HTTPRequest
from tornado.options import define, options
from concurrent.futures import ThreadPoolExecutor
from tornado.concurrent import run_on_executor
import redis
from multiprocessing import Process, Queue
import numpy as np
import cv2
import base64
import glob
from skimage import io

# TODO
# 1. logging format
# 2. make redis async

PRODUCT_URL = "http://192.168.1.11:8000/product"
FASHION_URL = "http://192.168.1.11:8000/fashion"
LOGO_URL = "http://192.168.1.11:8000/logo"
RERANK_URL = "http://192.168.1.11:8000/rerank"

"""
FASHION_URL = "http://localhost:9001/fashion"
LOGO_URL = "http://localhost:8003/logo"
RERANK_URL = "http://localhost:8004/rerank"
"""

PRICE_URL = "http://p.3.cn/prices/mgets?skuIds=%s"
QUERY_PRICE_ONLINE = True

TIMEOUT = 10
TOPN = 3

logging.basicConfig(level = logging.INFO,
                    format = '%(asctime)s %(filename)s[line:%(lineno)d] %(levelname)s %(message)s',
                    datefmt = '%a, %d %b %Y %H:%M:%S'
)

settings = {
    'debug': False,
}

define("port", default = 8889, help = "run on the given port", type = int)

# redis connection pool
REDIS_HOST = '192.168.1.11'
REDIS_PORT = 6379
REDIS_DB = 0
redis_conns = redis.Redis(host = REDIS_HOST, port = REDIS_PORT, db = REDIS_DB)

# utils for timing
def get_ts():
    return int(time.time()*1000)

# subprocess for saveing images
IMAGE_LOG_DIR = "/var/log/XXX_vision/frontend/images/"
def save(queue):
    print("Process for image saving is Started...")
    def _restore(img_bytes):
        imagedata = base64.b64decode(img_bytes)
        img_array = np.fromstring(imagedata, np.uint8)
        im = cv2.imdecode(img_array, cv2.IMREAD_COLOR)
        im = im[:,:,::-1]
        return im
    while True:
        try:
            randomValue, img_bytes = queue.get()
            img = _restore(img_bytes)
            current_ts, current_date = get_ts(), str(datetime.date.today())
            dirname = os.path.join(IMAGE_LOG_DIR, current_date)
            if not os.path.exists(dirname):
                os.mkdir(dirname)
            filename = os.path.join(dirname, "rid_%s_ts_%d.jpg" % (randomValue, current_ts))
            io.imsave(filename, img)
        except:
            pass

queue = Queue(10000)
subproc = Process(target = save, args = (queue,))
subproc.start()

class MainHandler(tornado.web.RequestHandler):
    executor = ThreadPoolExecutor(20)

    #def write_error(self, status_code, **kwargs):
    #    self.write(json.dumps(self.__prepare_default_result()))

    def __prepare_default_result(self):
        res = {
            'code': "100",
            'msg': "",
            'data': [],
        }
        return res

    def __generate_requests(self):
        post_body= self.request.body
        product_request = HTTPRequest(url = PRODUCT_URL, method = "POST", request_timeout = TIMEOUT, body = post_body)
        fashion_request = HTTPRequest(url = FASHION_URL, method = "POST", request_timeout = TIMEOUT, body = post_body)
        logo_request = HTTPRequest(url = LOGO_URL, method = "POST", request_timeout = TIMEOUT, body = post_body)
        return product_request, fashion_request, logo_request

    def __generate_fake_requests(self, params):
        product_request = HTTPRequest(url = PRODUCT_URL, method = "GET", request_timeout = TIMEOUT)
        fashion_request = HTTPRequest(url = FASHION_URL, method = "GET", request_timeout = TIMEOUT)
        logo_request = HTTPRequest(url = LOGO_URL, method = "GET", request_timeout = TIMEOUT)
        return product_request, fashion_request, logo_request

    @run_on_executor
    def __add_product_attrs(self, raw_products):
        # TODO: make is async
        products = []
        obselete_pids = set()
        err = ""
        fields = ["name", "url", "price", "image"]
        try:
            with redis_conns.pipeline() as p:
                for pid, _ in raw_products:
                    p.hmget(pid, *fields)
                redis_result = p.execute()
        except Exception as tx:
            err = "Fail to query redis: %s" % str(tx)
            logging.error(err)
            return products, obselete_pids, err

        errs = []
        for product, pinfo in zip(raw_products, redis_result):
            pid, weight = product
            if pinfo is None:
                errs.append(pid)
                product_info = {}
                obselete_pids.add(pid)
            else:
                product_info = dict(zip(fields, pinfo))
            products.append([pid, weight, product_info])
        if errs:
            err = "Fail to fetch product info from redis: %s" % str(errs)
            logging.warn(err)
        return products, obselete_pids, err

    def __check_input(self, params):
        """
        if params['code'] != '100100':
            logging.error("param 'code' must be 100100: %s" % params['code'])
            return False
        """
        if not params['content']:
            logging.error("param 'content' cannot be empty")
            return False
        if params['db_id'] not in ('101', '102'):
            logging.error("param 'db_id' must be 101 or 102: %s" % params['db_id'])
            return False
        if not params['classid'].isdigit():
            logging.error("param 'classid' must be a number: %s" % params['classid'])
            return False
        classid = int(params['classid'])
        if classid < 0 or classid > 3:
            logging.error("param 'classid' must be in (0, 1, 2, 3): %s" % params['classid'])
            return False
        return True

    def __parse_price_result(self, price_result):
        # the status code is successful or not
        err = ""
        if price_result.code != 200:
            err = "status code is not 200 but %d" % price_result.code
            return {}, err

        # the body is json or not
        try:
            raw_data = json.loads(price_result.body)
        except Exception as tx:
            err = "Fail to get online price result: %s" % str(tx)
            raw_data = None
        if not raw_data:
            return {}, err

        # parse it to price_dict
        price_dict = {}
        errs = []
        for item in raw_data:
            pid, price = item.get("id"), item.get("p")
            if not pid or not price:
                errs.append("Nothing Got: %s" % str([pid, price]))
                continue
            if not pid.startswith('J_') or not pid[2:].strip():
                errs.append("Invalid PID: %s" % str([pid, price]))
                continue
            pid = "JD-" + pid[2:].strip()
            try:
                price = float(price)
            except Exception as tx:
                errs.append("Invalid Price: %s" % str([price, str(tx)]))
                continue
            price_dict[pid] = price
        return price_dict, errs

    def __generate_price_request(self, products):
        if not QUERY_PRICE_ONLINE:
            return None
        pids = [pid[3:] for pid, _, _ in products if pid.startswith("JD-") and pid[3:].isdigit()]
        if not pids:
            return None
        params = ",".join(pids)
        url = PRICE_URL % params
        return HTTPRequest(url = url, method = "GET", request_timeout = TIMEOUT)

    def __generate_rerank_request(self, randomValue, products, fashion_info, logo_info, classid):
        candidates = [(pid, weight) for pid, weight, _ in products]
        attrs = {
            'fashion': fashion_info,
            'logo': logo_info
        }
        body = urlencode({"candidates": json.dumps(candidates), "attrs": json.dumps(attrs), 'classid': classid, 'randomValue': randomValue})
        return HTTPRequest(url = RERANK_URL, method = "POST", request_timeout = TIMEOUT, body = body)

    def __assemble_result(self, pids, products, online_price_dict, res):
        def _generate_product_dict(products):
            product_dict = {}
            for product in products:
                pid, _, _ = product
                product_dict[pid] = product
            return product_dict

        def _select_price(db_price, online_price):
            if online_price is not None and online_price > 0:
                return online_price
            if db_price is not None and float(db_price) > 0:
                return float(db_price)
            return None

        product_dict = _generate_product_dict(products)
        for pid in pids:
            _, _, attrs = product_dict[pid]
            if not attrs:
                continue
            price = _select_price(attrs.get('price'), online_price_dict.get('price'))
            if not price:
                continue
            product_item = {
                    'gid': pid,
                    'name': attrs['name'].decode('utf-8'),
                    'image': attrs['image'].decode('utf-8'),
                    'url': attrs['url'].decode('utf-8'),
                    'price': price,
                    'from': pid.split('-')[0],
                    'item': [],
            }
            res['data'].append(product_item)

        # mark as update if there is any data
        if res['data']:
            res['code'] = 101


    def __parse_product_result(self, product_result):
        products = None
        err = ""
        if product_result.code == 200:
            try:
                products = json.loads(product_result.body)
            except Exception as tx:
                err = "Result from product service is invalid (%s): %s" % (str(tx), product_result.body)
        else:
            err = "Product Request is failed: %s" % str(product_result)
        #logging.debug("Input: %s, Output: %s" % (product_result, products))
        return products, err


    def __parse_fashion_result(self, fashion_result):
        fashion_info = []
        err = ""
        if fashion_result.code == 200:
            try:
                fashion_info = json.loads(fashion_result.body)
            except Exception as tx:
                err = "Result from fashion service is invalid(%s): %s" % (str(tx), fashion_result.body)
        else:
            err = "Fashion Request is failed: %s" % str(fashion_result)
        logging.debug("Input: %s, Output: %s" % (fashion_result, fashion_info))
        return fashion_info, err


    def __parse_logo_result(self, logo_result):
        logo_info = []
        err = ""
        if logo_result.code == 200:
            try:
                logo_info = json.loads(logo_result.body)
            except Exception as tx:
                err = "Result from logo service is invalid(%s): %s" % (str(tx), logo_result.body)
        else:
            err = "LOGO Request is failed: %s" % str(logo_result)
        logging.debug("Input: %s, Output: %s" % (logo_result, logo_info))
        return logo_info, err


    def __retrieve_input_params(self):
        params = {}
        err = ""
        randomValue = self.get_body_argument("randomValue", None)
        if not randomValue:
            err = "randomValue is not set"
            return params, err
        try:
            #code, content, db_id, classid = '100100', '123', '101', '0'
            code = self.get_body_argument('code').strip()
            content = self.get_body_argument("content")
            db_id = self.get_body_argument("db_id").strip()
            classid = self.get_body_argument("classid").strip()
            params = {
                'code': code,
                'content': content,
                'db_id': db_id,
                'classid': classid,
                'randomValue': randomValue,
            }
        except Exception as e:
            err = str(e)

        try:
            debug = int(self.get_body_argument("debug").strip())
        except:
            debug = 0
        params['debug'] = debug

        #logging.debug(params)
        return params, err


    def __parse_rerank_result(self, rerank_result):
        pids = None
        err = ""
        if rerank_result.code == 200:
            try:
                pids = json.loads(rerank_result.body)
            except Exception as tx:
                err = "Result from rerank service is invalid(%s): %s" % (str(tx), rerank_result.body)
        else:
            err = "Rerank Request is failed: %s" % str(rerank_result)
        logging.debug(pids)
        return pids, err


    @gen.coroutine
    def post(self):
        #logging.debug(self.request.body)

        err_msg, tc = {}, {}
        start_ts = get_ts()

        # default result
        res = self.__prepare_default_result()

        # get & validate input params
        params, err = self.__retrieve_input_params()
        err_msg['param'] = err
        if not params or not self.__check_input(params):
            logging.error("Fail to retrieve params: %s" % str(err_msg))
            self.write(json.dumps(res))
            return
        randomValue = params['randomValue']
        res['msg'] = randomValue
        param_ts = get_ts()
        tc['param'] = param_ts - start_ts

        # send requests to backend-services at the same time
        http_client = AsyncHTTPClient()
        product_request, fashion_request, logo_request = self.__generate_requests()
        product_result, fashion_result, logo_result = yield [http_client.fetch(product_request, raise_error = False), \
                                                            http_client.fetch(fashion_request, raise_error = False), \
                                                            http_client.fetch(logo_request, raise_error = False)]
        subsrv_ts = get_ts()
        tc['subsrv_req'] = [product_result.request_time, fashion_result.request_time, logo_result.request_time, subsrv_ts - param_ts]

        # product candidates must exist
        products, err = self.__parse_product_result(product_result)
        err_msg['product_parse'] = err
        if not products:
            logging.error("[RandomValue %s] Fail to parse products: %s" % (randomValue, str(err_msg)))
            try:
                queue.put((randomValue, params['content']), block = False, timeout = 0.01)
            except:
                logging.warn("Fail to save image for RandomValue: %s" % randomValue)
            self.write(json.dumps(res))
            return

        # fashion and logo could be empty
        fashion_info, err = self.__parse_fashion_result(fashion_result)
        err_msg['fashion_parse'] = err
        logo_info, err = self.__parse_logo_result(logo_result)
        err_msg['logo_parse'] = err
        parse_ts = get_ts()
        tc['subsrv_parse'] = parse_ts - subsrv_ts

        # extend product info from redis
        products_ext, obselete_pids, err = yield self.__add_product_attrs(products)
        err_msg['redis'] = err
        if not products_ext:
            logging.error("[RandomValue %s] Fail to access product info from redis: %s" % (randomValue, str(err_msg)))
            self.write(json.dumps(res))
            return
        redis_ts = get_ts()
        tc['ext_product'] = redis_ts - parse_ts

        # send rerank and price request at the same time
        rerank_tc = []
        rerank_request = self.__generate_rerank_request(randomValue, products_ext, fashion_info, logo_info, params['classid'])
        price_request = self.__generate_price_request(products_ext)
        if price_request:
            rerank_result, online_price_result = yield [http_client.fetch(rerank_request, raise_error = False), \
                                                        http_client.fetch(price_request, raise_error = False)]
            #print(rerank_result)
            #print(online_price_result)
            rerank_tc.extend([rerank_result.request_time, online_price_result.request_time])
            online_price_result, errs = self.__parse_price_result(online_price_result)
            err_msg['online_price'] = errs
        else:
            rerank_result = yield http_client.fetch(rerank_request, raise_error = False)
            rerank_tc.append(rerank_result.request_time)
            online_price_result = {}
        rerank_ts = get_ts()
        rerank_tc.append(rerank_ts - redis_ts)
        tc['rerank'] = rerank_tc

        # if rerank is failed, then use the raw results
        pids, err = self.__parse_rerank_result(rerank_result)
        err_msg['rerank'] = err

        ######################################
        if not pids:
            self.write(json.dumps(res))
            return
        ######################################

        if not pids:
            pids = [pid for pid, w, _ in products_ext]

        # filter out obselete products
        pids = [pid for pid in pids if pid not in obselete_pids]

        # get topN pids
        #top_pids = pids[:TOPN]

        logging.info("RandomValue: %s, ObPids: %s" % (randomValue, str(pids)))

        # assemble results
        #self.__assemble_result(top_pids, products_ext, online_price_result, res)
        self.__assemble_result(pids, products_ext, online_price_result, res)

        if not params.get('debug', 0):
            res['data'] = res['data'][:TOPN]

        self.write(json.dumps(res))
        assemble_ts = get_ts()
        tc['assemble'] = assemble_ts - rerank_ts

        """
        # image saving async
        try:
            queue.put((randomValue, params['content']), block = False, timeout = 0.01)
        except:
            logging.warn("Fail to save image for RandomValue: %s" % randomValue)
        """
        del params['content']

        end_ts = get_ts()
        tc['image'] = end_ts - assemble_ts
        tc['total'] = end_ts - start_ts

        logging.info("RandomValue: %s, Input: %s, Output: %s, Errs: %s, Time: %s" % (randomValue, params, res, err_msg, tc))



if __name__ == "__main__":

    options.parse_command_line()
    app = tornado.web.Application([
        (r"/shopping", MainHandler),
        (r"/shopping_test", MainHandler),
    ], **settings)
    logging.info("Start Server on Port %d" % options.port)
    app.listen(options.port)
    tornado.ioloop.IOLoop.current().start()


