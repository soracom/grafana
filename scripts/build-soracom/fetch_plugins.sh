cd "$(dirname "$0")"

node_version=$(node --version)
echo "Node.js version: $node_version"

#install this package in a throwaway dir so we can reuse it a few times
npm install --prefix ./local  @grafana/sign-plugin@latest -g
PLUGIN_DIR=./plugins

mkdir -p $PLUGIN_DIR
cd $PLUGIN_DIR

clone_private_repo () {
  if [ ! -z "$2" ]; then
    COMMIT=$2
  else
    echo "ERROR No commit specified for $1, exiting"
    exit 1
  fi
  
  if [ -d $1 ]; then
<<<<<<< HEAD
    cd $1
    git pull origin $BRANCH
=======
    cd $1 || exit
    git fetch origin $COMMIT
    git checkout FETCH_HEAD
>>>>>>> 48b1723427 ([build] First attempt at getting plugins by commit hash)
    cd ..
  else
    echo "$(pwd)"
    echo `ls ..`
    echo `ls ../deploy_keys`
    echo `ls ../deploy_keys/$1/`

    echo "export GIT_SSH_COMMAND=\"ssh -i ../deploy_keys/$1/id_rsa -F /dev/null\""
    export GIT_SSH_COMMAND="ssh -i ../deploy_keys/$1/id_rsa -F /dev/null"

    echo "git init $1"
    git init $1

    echo "cd $1 || exit"
    cd $1 || exit

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

clone_private_repo soracom-harvest-backend e6260238d3e7afe55ef8bf9a5e88f836a58e9b99
clone_private_repo soracom-map-panel 59be62df090b858cad049b64db5527d9d8c5ef05
clone_private_repo soracom-image-panel a3385ba1e6507cb8cc7efff29fe96af1b55b10f5
clone_private_repo soracom-plot-panel d67d734f0c36e8ecde7e0a7cdecda4784178a497
clone_private_repo soracom-dynamic-image-panel 4df052c511cd4161f709ba15b3c5912bbb5c0040

#Add any pre-built plugins to the dir
cp -R ../pre-built-plugins/* .
