demo:
	bazel build //cmd/imagebuilder
	bazel-bin/cmd/imagebuilder/imagebuilder fetch docker://busybox --v=4 --alsologtostderr
	bazel-bin/cmd/imagebuilder/imagebuilder create layer layer1 --base docker://busybox
	date > info
	bazel-bin/cmd/imagebuilder/imagebuilder cp info layer1:/info
	bazel-bin/cmd/imagebuilder/imagebuilder set layer1 cmd cat /info
	bazel-bin/cmd/imagebuilder/imagebuilder push layer1 docker://justinsb/test --v=4 --alsologtostderr

gofmt:
	gofmt -w -s pkg/ cmd/

