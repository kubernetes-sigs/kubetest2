.PHONY: install
install:
	GO111MODULE=on go install .

.PHONY: install-deployer-%
install-deployer-%:
	GO111MODULE=on go install ./kubetest2-$*

.PHONY: quick-verify
quick-verify: install install-deployer-kind
	kubetest2 kind --up --down --test=exec -- kubectl get all -A
