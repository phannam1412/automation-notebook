#!/bin/bash

go get -u gopkg.in/yaml.v2
go get -u github.com/PuerkitoBio/goquery
go get -u github.com/fsnotify/fsnotify
go get -u github.com/tealeg/xlsx
go get -u github.com/yudai/gojsondiff
touch history.txt
echo "no history, please search and run some commands" >> history.txt