.PHONY: all
all:
	bazel build //cmd/...

.PHONY: demo
demo:
	bazel build //cmd/kcb
	bazel-bin/cmd/kcb/kcb fetch docker://busybox --v=4 --alsologtostderr
	bazel-bin/cmd/kcb/kcb create layer layer1 --base docker://busybox
	date > info
	bazel-bin/cmd/kcb/kcb cp info layer1:/info
	bazel-bin/cmd/kcb/kcb set layer1 cmd cat /info
	bazel-bin/cmd/kcb/kcb push layer1 docker://justinsb/test --v=4 --alsologtostderr

.PHONY: gofmt
gofmt:
	gofmt -w -s pkg/ cmd/

.PHONY: goimports
goimports:
	goimports -w cmd/ pkg/

.PHONY: test
test:
	bazel test //... --test_output=streamed

.PHONY: gazelle
gazelle:
	bazel run //:gazelle

.PHONY: push-to-github
push-to-github:
	bazel build --experimental_platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //cmd/...
	bazel build --experimental_platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64 //cmd/...
	shipbot -tag "${SHIPBOT_VERSION}" -config .shipbot.yaml
