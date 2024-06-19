#!/bin/bash

postid=$1
curl -v -b ./cookie -X PUT localhost:8000/posts/liked/$postid

