#!/bin/bash
set -e

cd "$(dirname "$0")" || exit

node_version=$(node --version)
echo "Node.js version: $node_version"

#install this package in a throwaway dir so we can reuse it a few times
npm install --prefix ./local  @grafana/sign-plugin@latest -g
PLUGIN_DIR=./plugins

mkdir -p $PLUGIN_DIR
cd $PLUGIN_DIR || exit

clone_private_repo () {
  if [ ! -z "$2" ]; then
    echo "Cloning $1 at commit $2"
    COMMIT=$2
  else
    echo "ERROR No commit specified for $1, exiting"
    exit 1
  fi

  # Suppress message about initial branch name
  # Suppress message about detached head
  git config --global init.defaultBranch main
  git config --global advice.detachedHead false
  
  if [ -d $1 ]; then
    cd $1 || exit
    echo "git fetch origin $COMMIT"
    git fetch origin $COMMIT

    echo "git checkout FETCH_HEAD"
    git checkout FETCH_HEAD
    cd ..
  else
    echo "$(pwd)"
    echo `ls ..`
    echo `ls ../deploy_keys`
    echo `ls ../deploy_keys/$1/`

    echo "git init $1"
    git init $1

    echo "cd $1 || exit"
    cd $1 || exit

    echo "export GIT_SSH_COMMAND=\"ssh -i ../../deploy_keys/$1/id_rsa -F /dev/null\""
    export GIT_SSH_COMMAND="ssh -i ../../deploy_keys/$1/id_rsa -F /dev/null"

    echo "git remote add origin git@github.com:soracom/$1.git"
    git remote add origin git@github.com:soracom/$1.git

    echo "git fetch origin $COMMIT"
    git fetch origin $COMMIT

    echo "git checkout FETCH_HEAD"
    git checkout FETCH_HEAD

    echo "cd .."
    cd ..
  fi

  if [ -f "$1/signplugin.sh" ]; then 
    cd $1 || exit
    ./signplugin.sh || exit 1
    cd ..
  fi
}

clone_public_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2 || exit
    git pull origin master
    cd ..
  else
    git clone https://github.com/$1/$2.git $1-$2
  fi
  if [ ! -z "$3" ]; then
    cd $1-$2 || exit
    echo "Checking out specific commit $3"
    git checkout $3
    cd ..
  fi
}

build_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2 || exit
    npm install
    npm run-script build
    cd ..
  fi
}

yarn_build_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2 || exit
    npm install yarn
    npx yarn install --pure-lockfile
    npx yarn build
    cd ..
  fi
}

clone_private_repo soracom-harvest-backend 6820e212a34608e47f3766ff9e059b6ed2c8ca5b
clone_private_repo soracom-map-panel 59be62df090b858cad049b64db5527d9d8c5ef05
clone_private_repo soracom-image-panel a3385ba1e6507cb8cc7efff29fe96af1b55b10f5
clone_private_repo soracom-plot-panel a166c5f3da64896d6ac6a2ddc39b4551dbc5c9c3
clone_private_repo soracom-dynamic-image-panel 4df052c511cd4161f709ba15b3c5912bbb5c0040

#Add any pre-built plugins to the dir
cp -R ../pre-built-plugins/* .
