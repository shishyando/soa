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

class TestSocialNetworkMethods(unittest.TestCase):

    @classmethod
    def setUpClass(self):
        self.host = "http://localhost:8000/"
        self.addrs = {
            Handles.REGISTER: self.host + "users/register",
            Handles.LOGIN: self.host + "users/login",
            Handles.UPDATE_USER: self.host + "users",
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
            "email": "qwe@mail.ru",
            "name": "Pavel",
            "surname": "Shishikhin",
            "dateOfBirth": "20.01.2004",
            "phoneNumber": "+71230005577"
        }), cookies=cookies.get_dict())
        self.assertEqual(r.status_code, 200)


if __name__ == '__main__':
    unittest.main()
