from pprint import pprint
import unittest
import requests
from enum import Enum
import uuid
import json

class Handles(Enum):
    REGISTER = 1
    LOGIN = 2
    UPDATE_USER = 3
    POST_CREATE = 4
    POST_UPDATE = 5
    POST_DELETE = 6
    POST_GET = 7
    PAGE_GET = 8

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
            Handles.POST_DELETE: self.host + "posts/delete",
            Handles.POST_GET: self.host + "posts/single",
            Handles.PAGE_GET: self.host + "posts/page",
        }
        self.login = uuid.uuid4().hex[:7].upper()
        self.password = uuid.uuid4().hex[:7].upper()


        # register once
        r = requests.post(self.addrs[Handles.REGISTER], data=json.dumps({
            "login": self.login,
            "password": self.password
        }))


    def try_login(self):
        r = requests.post(self.addrs[Handles.LOGIN], data=json.dumps({
            "login": self.login,
            "password": self.password
        }))
        self.assertEqual(r.status_code, 200)
        jwt_cookie = r.cookies.get("jwt")
        self.assertIsNotNone(jwt_cookie)
        return r.cookies


    def test_login(self):
        self.try_login()


    def test_update(self):
        cookies = self.try_login()

        # update
        r = requests.put(self.addrs[Handles.UPDATE_USER], data=json.dumps({
            "login": self.login,
            "password": self.password,
            "email": "qwe@mail.ru"
        }), cookies=cookies.get_dict())
        self.assertEqual(r.status_code, 200)


    def test_post(self):
        cookies = self.try_login()

        r = requests.put(self.addrs[Handles.POST_CREATE], data=json.dumps({
            "Title": "post#1",
            "Content": "some contents",
            "AuthorLogin": self.login,
        }), cookies=cookies.get_dict())

        self.assertEqual(r.status_code, 200)
        postId = int(r.json()["PostId"])

        r = requests.get(self.addrs[Handles.POST_GET], data=json.dumps({
            "PostId": postId
        }), cookies=cookies.get_dict())
        self.assertEqual(r.status_code, 200)
        self.assertIn("post#1", r.text)

        r = requests.put(self.addrs[Handles.POST_UPDATE], data=json.dumps({
            "PostId": postId,
            "Title": "post#0 (updated)",
            "Content": "brand new data",
            "AuthorLogin": self.login
        }), cookies=cookies.get_dict())
        self.assertEqual(r.status_code, 200)

        r = requests.get(self.addrs[Handles.POST_GET], data=json.dumps({
            "PostId": postId
        }), cookies=cookies.get_dict())
        self.assertEqual(r.status_code, 200)
        self.assertIn("post#0 (updated)", r.text)

        r = requests.put(self.addrs[Handles.POST_CREATE], data=json.dumps({
            "Title": "post#2",
            "Content": "abacaba",
            "AuthorLogin": self.login,
        }), cookies=cookies.get_dict())

        self.assertEqual(r.status_code, 200)
        postId = int(r.json()["PostId"])

        r = requests.get(self.addrs[Handles.PAGE_GET], data=json.dumps({
            "PageId": 0
        }), cookies=cookies.get_dict())

        pprint(r.json())
        self.assertEqual(r.status_code, 200)
        self.assertIn("post#0 (updated)", r.text)
        self.assertIn("post#2", r.text)


if __name__ == '__main__':
    unittest.main()
