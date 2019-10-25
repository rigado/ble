native: clean
	go build -o hci-test

arm: clean
	env GOOS=linux GOARCH=arm go build -o hci-test_arm
	# scp hci-test_arm admin@192.168.2.107:~

clean:
	rm -f hci-test hci-test_arm
