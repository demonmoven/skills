mkdir -p /root/log || true

set -ex

echo "[$(date +%Y-%m-%dT%H:%M:%S)]enter entrypoint" >> /root/log/bootstrap.log
echo "[$(date +%Y-%m-%dT%H:%M:%S)]start build" >> /root/log/bootstrap.log

sh build.sh

echo "[$(date +%Y-%m-%dT%H:%M:%S)]finish build" >> /root/log/bootstrap.log

cd output && pwd && sh bootstrap.sh