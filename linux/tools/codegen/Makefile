signal_out="../../hci/signal_gen.go"
cmd_out="../../hci/cmd/cmd_gen.go"
evt_out="../../hci/evt/evt_gen.go"
att_out="../../att/att_gen.go"

targets := signal cmd evt att

all: ${targets}

${targets}:
	go run codegen.go -tmpl $@ -out ${$@_out} && goimports -w ${$@_out}

