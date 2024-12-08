.DEFAULT_GOAL := help

## 変えるところ
SERVICE:=isuride-go.service
BUILD_APP:= cd go; go build -o isuride *.go # go build -o isux *.go など
GITHUB_REMOTE_URL:=git@github.com:machida4/isucon14.git

## 定数

MYSQL_SLOW_LOG:=/var/log/mysql/mysql-slow.log
NGINX_LOG:=/var/log/nginx/access.log
NGINX_ERROR_LOG:=/var/log/nginx/error.log

ALP_LOG:=alp.txt
DIGEST_LOG:=digest.txt

DISCORD_WEBHOOK_URL:=${DISCORD_WEBHOOK_URL}

define MYSQL_CONFIG
[mysqld]
slow_query_log=ON
long_query_time=0
slow_query_log_file=/var/log/mysql/mysql-slow.log
endef
export MYSQL_CONFIG

## Common

restart: ## Restart all
	@git pull
	@make -s nginx-restart
	@make -s db-restart
	@make -s app-restart

report: ## Generate monitoring report
	@make -s alp
	@make -s digest

restart-1: ## Restart for Server 1
	@make -s restart

restart-2: ## Restart for Server 2
	@make -s restart

restart-3: ## Restart for Server 3
	@make -s restart

## App

app-restart: ## Restart Server
	$(BUILD_APP)
	@sudo systemctl daemon-reload
	@sudo systemctl restart $(SERVICE)
	@echo 'Restart service'

app-log: ## Tail server log
	@sudo journalctl -f -n10 -u $(SERVICE)

## Nginx

nginx-restart: ## Restart nginx
	@sudo cp /dev/null $(NGINX_LOG)
	@sudo cp -aT ./nginx /etc/nginx/
	@echo 'Validate nginx.conf'
	@sudo nginx -t
	@sudo systemctl restart nginx
	@echo 'Restart nginx'

nginx-log: ## Tail nginx access.log
	@sudo tail -f $(NGINX_LOG)

nginx-error-log: ## Tail nginx error.log
	@sudo tail -f $(NGINX_ERROR_LOG)

## Alp
# 1xxを除いて少し並べかえる
alp: ## Run alp
	sudo alp ltsv --file $(NGINX_LOG) --sort sum --reverse \
	--matching-groups='/posts/\d+, /@\w+, /image/\d+.\w{3}' \
	-o 'count,2xx,3xx,4xx,5xx,method,uri,sum,avg,min,max,p90,p95,p99,stddev,min_body,max_body,sum_body,avg_body'\
	> $(ALP_LOG)
	echo $(DISCORD_WEBHOOK_URL)
	@DISCORD_WEBHOOK_URL=$(DISCORD_WEBHOOK_URL) ./dispost -f $(ALP_LOG)

## DB

db-restart: ## Restart mysql
	@sudo cp /dev/null $(MYSQL_SLOW_LOG)
	-@sudo cp -L my.cnf /etc/mysql/
	@sudo systemctl restart mysql
	@echo 'Restart mysql'

digest: ## Analyze mysql-slow.log by pt-query-digest
	@sudo pt-query-digest $(MYSQL_SLOW_LOG) > $(DIGEST_LOG)
	@DISCORD_WEBHOOK_URL=$(DISCORD_WEBHOOK_URL) ./dispost -f $(DIGEST_LOG)

## profile
prof: ## pprofとfgprofで記録
	-process-compose down
	sleep 1
	process-compose up -D prof pprof-server fgprof-server

## etc

.PHONY: log
log: ## Tail journalctl
	@sudo journalctl -f

show-running-services: ## Show running systemctl service units
	@sudo systemctl list-units --type=service --state=running

.PHONY: help
help:
	@grep -E '^[a-z0-9A-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

## Setup
.PHONY: setup-base, setup, setup-alp, setup-slow-query, setup-git, setup-shell, setup-process-compose

setup: dispost setup-alp setup-slow-query setup-git setup-shell setup-process-compose ## Setup

setup-base: ## Setup base
	@sudo apt install -y unzip curl graphviz memcached
	mkdir -p ~/.ssh
	@grep ssh- ~/.ssh/authorized_keys || curl -s https://github.com/{machida4,NosKmc,laft2}.keys >> ~/.ssh/authorized_keys

dispost: setup-base ## Setup dispost
	@wget https://raw.githubusercontent.com/machida4/dispost/refs/heads/master/dispost
	@sudo chmod 766 dispost

setup-alp: setup-base ## Setup alp
	@wget https://gist.githubusercontent.com/machida4/83b84fdc4f39f2066def530e473c9231/raw/b1f8b9445a3525d5a14a8e5edad1ab0a3b756254/install_alp.sh
	@sudo chmod u+x install_alp.sh
	./install_alp.sh
	rm install_alp.sh
	rm alp_linux_amd64.zip

setup-slow-query: setup-base ## Setup pt-query-digest
	@wget https://gist.githubusercontent.com/machida4/83b84fdc4f39f2066def530e473c9231/raw/b1f8b9445a3525d5a14a8e5edad1ab0a3b756254/install_pt-query-digest.sh
	@chmod u+x install_pt-query-digest.sh
	./install_pt-query-digest.sh	

	@echo "create mysql-slow.log"
	@sudo touch $(MYSQL_SLOW_LOG)
	@sudo chmod 777 $(MYSQL_SLOW_LOG)
	@echo "mysql-slow.log is created"

	@echo "setup my.cnf"
	echo "$${MYSQL_CONFIG}" | sudo tee -a my.cnf
	@make -s db-restart

	rm install_pt-query-digest.sh
	rm v3.3.1.tar.gz

setup-git: ## Setup git
	which git || sudo apt install -y git
	git config --global user.email isucon@example.com
	git config --global user.name isucon
	@if ! [ -f ~/.ssh/id_ed25519 ]; then\
		mkdir -p ~/.ssh;\
		ssh-keygen -t ed25519 -N "" -f "$${HOME}/.ssh/id_ed25519";\
		echo "register ~/.ssh/id_ed25519.pub to github.com";\
	fi
	sudo ls ./nginx 2>/dev/null || sudo cp -a /etc/nginx ./nginx
	sudo chown -R isucon:isucon ./nginx
	sudo cp -L /etc/mysql/my.cnf ./my.cnf
	sudo chown isucon:isucon ./my.cnf
	git init

setup-process-compose: setup-base ## Setup process-compose
	sudo sh -c "$$(curl --location https://raw.githubusercontent.com/F1bonacc1/process-compose/main/scripts/get-pc.sh)" -- -d -b /usr/local/bin
	grep process-compose ~/.bash_completion > /dev/null 2>/dev/null || process-compose completion bash >> ~/.bash_completion
	grep PC_PORT_NUM ~/.bashrc || echo 'export PC_PORT_NUM=11649' >> ~/.bashrc
	wget https://gist.githubusercontent.com/laft2/8308938911654304ecfe83cfc261583d/raw/2aae258c784b771950eb5085f81eee707f599873/process-compose.yaml -O ./process-compose.yaml

define GIT_PS1
source ~/.git-completion.bash
source ~/.git-prompt.sh
GIT_PS1_SHOWDIRTYSTATE=true
GIT_PS1_SHOWUNTRACKEDFILES=true
GIT_PS1_SHOWSTASHSTATE=true
GIT_PS1_SHOWUPSTREAM=auto
PS1="\[\\e[1;32m\]\u@\h\[\\e[m\]:\[\\e[1;34m\]\W\[\\e[m\]\[\\e[33m\]\$$(__git_ps1)\[\\e[m\]\$$ "
endef
export GIT_PS1

setup-shell: ## Setup shell
	wget https://raw.githubusercontent.com/git/git/master/contrib/completion/git-completion.bash -O ~/.git-completion.bash
	wget https://raw.githubusercontent.com/git/git/master/contrib/completion/git-prompt.sh -O ~/.git-prompt.sh
	echo "$${GIT_PS1}" > .git_ps1.bash
	grep git_ps1.bash ~/.bashrc || echo 'source "$(CURDIR)/.git_ps1.bash"' >> ~/.bashrc
	grep vim ~/.bashrc || echo 'export EDITOR=vim' >> ~/.bashrc
## get alp group (makefile内のalpに渡す時に$は$$でエスケープする必要があるので末尾の$を重ねる実装をしている)
#cat webapp/go/main.go | grep -E 'GET|POST|PUT|DELETE' | sed -E 's/.*\"(.+)\".*/\1$$/' | sed -E 's/:id/[0-9]+/' | perl -pe 's/\n/ , /g'

git-force-sync: ## n+1台目のサーバー等を強制的にmasterに同期させる用
	@git remote add origin $(GITHUB_REMOTE_URL)
	@git fetch origin
	@git reset --hard origin/master
	@git branch --set-upstream-to=origin/master master
