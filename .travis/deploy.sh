#!/bin/bash

eval "$(ssh-agent -s)"
chmod 600 .travis/id_rsa
ssh-add .travis/id_rsa

git config --global push.default matching
git remote add deploy ssh://$GITUSER@$IP:$PORT$DEPLOY_DIR
git push deploy master

ssh -tt $GITUSER@$IP -p $PORT <<EOF
	go get -u github.com/EnsicoinDevs/eccd
	sudo systemctl restart eccd.service
	exit
EOF
