all:
	sudo firewall-cmd --add-port=8080/tcp
	sudo firewall-cmd --reload
	cd ./src/ && go run *.go
