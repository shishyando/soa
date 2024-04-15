#!/bin/bash
curl -v -c cookie -b cookie -X POST --data '{"login":"a","password":"a"}' localhost:8000/users/login
