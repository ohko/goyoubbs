#!/bin/sh

echo ===== 编译程序 =====
GOOS=linux GOARCH=amd64 go build -i -v -o goyoubbs -ldflags "-s -w" github.com/ohko/goyoubbs
if [ $? -ne 0 ];then echo "err!";exit 1;fi

echo ===== 制作docker =====
tar czf docker/tmp.tar.gz goyoubbs static config/config.yaml view
cd docker
docker build --no-cache -t registry.cdeyun.com/hk/goyoubbs .
cd ..
if [ $? -ne 0 ];then echo "err!";exit 1;fi

echo ===== 推送docker =====
docker push registry.cdeyun.com/hk/goyoubbs
if [ $? -ne 0 ];then echo "err!";exit 1;fi

echo ===== 清理 =====
docker images |grep "<none>"|awk '{print $3}'|xargs docker image rm
docker image rm registry.cdeyun.com/hk/goyoubbs
rm -rf docker/tmp.tar.gz
rm -rf goyoubbs

echo ===== 启动远程docker =====
ssh root@cdeyun.com "docker pull registry.cdeyun.com/hk/goyoubbs && docker rm -fv goyoubbs; docker images |grep '<none>'|awk '{print \$3}'|xargs docker image rm; docker run -d --restart=always --name goyoubbs -v /data/docker-goyoubbs:/data -v /data/docker-goyoubbs-upload:/static/upload -p 8088:8088 registry.cdeyun.com/hk/goyoubbs"
if [ $? -ne 0 ];then echo "err!";exit 1;fi