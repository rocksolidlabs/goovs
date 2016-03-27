current_dir := $(shell pwd)

unittest:
	@go test ./...

testimg:
	@cd docker && ./build.sh

targettest: testimg
	@docker run -itd --name ovs socketplane/openvswitch:latest
	@docker run -t --rm --link ovs:ovs -v $(current_dir):/go/src/github.com/kopwei/goovs goovstest:latest
	@docker rm -f ovs

.PHONY: testimg targettest unittest