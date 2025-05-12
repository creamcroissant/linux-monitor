#!/bin/bash

#安装依赖
apt update&&apt install zip unzip wget curl -y
apt-get install curl git mercurial make binutils bison gcc build-essential -y

# 下载Go
wget https://go.dev/dl/go1.20.linux-amd64.tar.gz

# 解压
sudo tar -C /usr/local -xzf go1.20.linux-amd64.tar.gz

# 创建软链接
sudo ln -s /usr/local/go/bin/* /usr/bin/

# 设置环境变量
echo 'export GOPATH="$HOME/go"' >> ~/.bashrc
echo 'export PATH="$PATH:/usr/local/go/bin:$GOPATH/bin"' >> ~/.bashrc

# 应用配置
source ~/.bashrc

# 检查版本
go version

# 清理下载的压缩包
rm go1.20.linux-amd64.tar.gz

echo "Go 安装完成!"

# 下载并安装 nvm
echo "正在下载并安装 nvm..."
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.2/install.sh | bash

# 加载 nvm，无需重启 shell
echo "加载 nvm..."
\. "$HOME/.nvm/nvm.sh"

# 下载并安装 Node.js
echo "正在下载并安装 Node.js..."
nvm install 23

#加载node环境
source ~/.bashrc

# 验证 Node.js 版本
echo "验证 Node.js 版本..."
node -v
echo "当前使用的 Node.js 版本："
nvm current

#安装npm
npm install -g npm

# 验证 npm 版本
echo "验证 npm 版本..."
npm -v

echo "安装完成！"
