import unittest
import requests
from enum import Enum
import uuid
import json
import time

class Handles(Enum):
    REGISTER = 1
    LOGIN = 2
    UPDATE_USER = 3
    POST_CREATE = 4
    POST_UPDATE = 5
    POST_DELETE = 6
    POST_GET = 7
    PAGE_GET = 8
    POST_VIEW = 9
    POST_LIKE = 10


def pprint_response(r: requests.Response):
    print(r.request.method, r.request.path_url, r.status_code)
    if len(r.content) and len(r.content.decode()):
        try:
            print("Response body:", json.dumps(json.loads(r.content.decode()), indent=4, sort_keys=True), sep='\n', end='\n\n')
        except:
            print("Response body:", r.content.decode(), sep='\n', end='\n\n')
    print("=" * 100)

class TestSocialNetworkMethods(unittest.TestCase):

    @classmethod
    def setUpClass(self):
        self.host = "http://localhost:8000/"
        self.addrs = {
            Handles.REGISTER: self.host + "users/register",
            Handles.LOGIN: self.host + "users/login",
            Handles.UPDATE_USER: self.host + "users",
            Handles.POST_CREATE: self.host + "posts/create",
            Handles.POST_UPDATE: self.host + "posts/update",
            Handles.POST_DELETE: self.host + "posts/delete/",
            Handles.POST_GET: self.host + "posts/single/",
            Handles.PAGE_GET: self.host + "posts/page/",
            Handles.POST_VIEW: self.host + "posts/viewed/",
            Handles.POST_LIKE: self.host + "posts/liked/",
        }
        self.login = uuid.uuid4().hex[:7].upper()
        self.password = uuid.uuid4().hex[:7].upper()

        healthy = False
        for _ in range(10):
            r = requests.get("http://localhost:8001/health")
            if r.status_code == 200:
                healthy = True
                break
            else:
                time.sleep(1)

        if not healthy:
            print("Unhealthy stats service! Exiting...")
            exit(1)

        # register once
        data = {
            "login": self.login,
            "password": self.password
        }
        r = requests.post(self.addrs[Handles.REGISTER], data=json.dumps(data))
        pprint_response(r)


    def try_login(self):
        data = {
            "login": self.login,
            "password": self.password
        }
        r = requests.post(self.addrs[Handles.LOGIN], data=json.dumps(data))
        pprint_response(r)
        self.assertEqual(r.status_code, 200)
        jwt_cookie = r.cookies.get("jwt")
        self.assertIsNotNone(jwt_cookie)
        return r.cookies


    def test_login(self):
        self.try_login()


    def test_update(self):
        cookies = self.try_login()

        # update
        data = {
            "login": self.login,
            "password": self.password,
            "email": "qwe@mail.ru"
        }
        r = requests.put(self.addrs[Handles.UPDATE_USER], data=json.dumps(data), cookies=cookies.get_dict())
        pprint_response(r)
        self.assertEqual(r.status_code, 200)


    def test_post(self):
        cookies = self.try_login()

        data = {
            "Title": "post#1",
            "Content": "some contents",
            "AuthorLogin": self.login,
        }
        r = requests.post(self.addrs[Handles.POST_CREATE], data=json.dumps(data), cookies=cookies.get_dict())
        pprint_response(r)

        self.assertEqual(r.status_code, 200)
        postId = int(r.json()["PostId"])

        r = requests.get(self.addrs[Handles.POST_GET] + str(postId), cookies=cookies.get_dict())
        pprint_response(r)
        self.assertEqual(r.status_code, 200)
        self.assertIn("post#1", r.text)

        data = {
            "PostId": postId,
            "Title": "post#0 (updated)",
            "Content": "brand new data",
            "AuthorLogin": self.login
        }
        r = requests.put(self.addrs[Handles.POST_UPDATE], data=json.dumps(data), cookies=cookies.get_dict())
        pprint_response(r)
        self.assertEqual(r.status_code, 200)

        r = requests.get(self.addrs[Handles.POST_GET] + str(postId), cookies=cookies.get_dict())
        pprint_response(r)
        self.assertEqual(r.status_code, 200)
        self.assertIn("post#0 (updated)", r.text)

        data = {
            "Title": "post#2",
            "Content": "abacaba",
            "AuthorLogin": self.login,
        }
        r = requests.post(self.addrs[Handles.POST_CREATE], data=json.dumps(data), cookies=cookies.get_dict())
        pprint_response(r)

        self.assertEqual(r.status_code, 200)
        postId = int(r.json()["PostId"])

        r = requests.get(self.addrs[Handles.PAGE_GET] + "0", cookies=cookies.get_dict())
        pprint_response(r)


        self.assertEqual(r.status_code, 200)
        self.assertIn("post#0 (updated)", r.text)
        self.assertIn("post#2", r.text)

    def view_post(self, postId: int):
        cookies = self.try_login()
        r = requests.put(self.addrs[Handles.POST_VIEW] + str(postId), cookies=cookies.get_dict())
        pprint_response(r)
        self.assertEqual(r.status_code, 200)
        time.sleep(1)

    def like_post(self, postId: int):
        cookies = self.try_login()
        r = requests.put(self.addrs[Handles.POST_LIKE] + str(postId), cookies=cookies.get_dict())
        pprint_response(r)
        self.assertEqual(r.status_code, 200)
        time.sleep(1)

    def get_post_stats(self, postId: int):
        r = requests.get(f"http://localhost:8001/cheat/{postId}")
        pprint_response(r)

        self.assertEqual(r.status_code, 200)
        stats = json.loads(r.content.decode())
        return stats['postId'], stats['views'], stats['likes']

    def test_stats(self):
        cookies = self.try_login()

        data = {
            "Title": "post_to_like",
            "Content": "very likable",
            "AuthorLogin": self.login,
        }
        r = requests.post(self.addrs[Handles.POST_CREATE], data=json.dumps(data), cookies=cookies.get_dict())
        pprint_response(r)

        self.assertEqual(r.status_code, 200)
        postId = int(r.json()["PostId"])

        self.view_post(postId)

        statsPostId, views, likes = self.get_post_stats(postId)
        self.assertEqual(statsPostId, postId)
        self.assertEqual(views, 1)
        self.assertEqual(likes, 0)

        self.view_post(postId)

        statsPostId, views, likes = self.get_post_stats(postId)
        self.assertEqual(statsPostId, postId)
        self.assertEqual(views, 2)
        self.assertEqual(likes, 0)

        self.like_post(postId)
        statsPostId, views, likes = self.get_post_stats(postId)
        self.assertEqual(statsPostId, postId)
        self.assertEqual(views, 2)
        self.assertEqual(likes, 1)

if __name__ == '__main__':
    unittest.main()
