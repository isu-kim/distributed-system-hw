for i in {1..6}; do
	docker run --rm -d -e LISTEN_ADDR=0.0.0.0 -e LISTEN_PORT=4480 -e LB_ADDR=172.17.0.2 -e LB_PORT=8080 --name tcp-echo-server-$i isukim/ds-hw-1-tcp-server
done
