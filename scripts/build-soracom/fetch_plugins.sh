set -e

cd "$(dirname "$0")" || exit

node_version=$(node --version)
echo "Node.js version: $node_version"

#install this package in a throwaway dir so we can reuse it a few times
npm install --prefix ./local  @grafana/sign-plugin@latest -g
PLUGIN_DIR=./plugins

mkdir -p $PLUGIN_DIR
cd $PLUGIN_DIR

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

  if [ ! -z $1/signplugin.sh ]; then 
    cd $1
    ./signplugin.sh || exit 1
    cd ..
  fi
}

clone_public_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2
    git pull origin master
    cd ..
  else
    git clone https://github.com/$1/$2.git $1-$2
  fi
  if [ ! -z "$3" ]; then
    cd $1-$2
    echo "Checking out specific commit $3"
    git checkout $3
    cd ..
  fi
}

build_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2
    npm install
    npm run-script build
    cd ..
  fi
}

yarn_build_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2
    npm install yarn
    npx yarn install --pure-lockfile
    npx yarn build
    cd ..
  fi
}

clone_private_repo soracom-harvest-backend HASH_PLACEHOLDER
clone_private_repo soracom-map-panel HASH_PLACEHOLDER
clone_private_repo soracom-image-panel HASH_PLACEHOLDER
clone_private_repo soracom-plot-panel HASH_PLACEHOLDER
clone_private_repo soracom-dynamic-image-panel HASH_PLACEHOLDER

#Add any pre-built plugins to the dir
cp -R ../pre-built-plugins/* .
