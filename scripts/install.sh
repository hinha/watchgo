VERSION="$1"

PATH="$PATH:/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
TARGET_DIR=/usr/local/bin/gowatch
CONF_DIR=/etc/gowatch
LOG_DIR=/var/log/gowatch
PERM="chmod +x /usr/local/bin/gowatch"

if [ `getconf LONG_BIT` = "32" ]; then
    ARCH="386"
else
    ARCH="amd64"
fi

URL="https://github.com/hinha/watchgo/releases/download/$VERSION/watchgo-$ARCH"
CONF_URL="https://raw.githubusercontent.com/hinha/watchgo/main/config.yml"

if [ -n "`which curl`" ]; then
    download_cmd="curl -L $URL --output $TARGET_DIR"
    conf_download_cmd="curl -L $CONF_URL --output $CONF_DIR/config.yml"
else
    die "Failed to download watchgo: curl not found, plz install curl"
fi

mkdir -p $CONF_DIR $LOG_DIR

echo -n "Fetching watchgo from $URL: "
$download_cmd || die "Error when downloading watchgo from $URL"
$conf_download_cmd || die "Error when downloading config file watchgo from $CONF_URL"
/bin/echo -e "Install watchgo: \x1B[32m done \x1B[0m"

echo -n "Set permission execute watchgo: "
$PERM || die "Error permission execut watchgo from $TARGET_DIR"
/bin/echo -e "\x1B[32m done \x1B[0m"
watchgo -v
/bin/echo -e "\x1B[32m Finished \x1B[0m"
