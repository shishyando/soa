#!/bin/bash
curl -v -b -i cookie -c cookie -X POST --data '{"login":"a","password":"a"}' localhost:8000/users/register
