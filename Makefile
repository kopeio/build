all:
	bazel build //...

demo:
	bazel build //cmd/kcb
	bazel-bin/cmd/kcb/kcb fetch docker://busybox --v=4 --alsologtostderr
	bazel-bin/cmd/kcb/kcb create layer layer1 --base docker://busybox
	date > info
	bazel-bin/cmd/kcb/kcb cp info layer1:/info
	bazel-bin/cmd/kcb/kcb set layer1 cmd cat /info
	bazel-bin/cmd/kcb/kcb push layer1 docker://justinsb/test --v=4 --alsologtostderr

gofmt:
	gofmt -w -s pkg/ cmd/

