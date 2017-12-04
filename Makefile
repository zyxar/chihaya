GO          = vgo
PRODUCT     = chihaya
GOARCH     := amd64
# GOLINT      = $(GOPATH)/bin/golint
# VERSION    := $(shell git describe --all --always --dirty --long)
# BUILD_TIME := $(shell date +%FT%T%z)
# LDFLAGS     = -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

all: $(shell $(GO) env GOOS)

build-%:
	$(eval $@_OS := $*)
	env GOOS=$($@_OS) GOARCH=$(GOARCH) $(GO) build ${LDFLAGS} -v -o $(PRODUCT)$(EXT) ./cmd/chihaya

linux: EXT=.elf
linux: build-linux

darwin: EXT=.mach
darwin: build-darwin

.PHONY: clean
clean:
	@rm -f $(PRODUCT) $(PRODUCT).elf $(PRODUCT).mach

# .PHONY: lint
# lint:
# 	@$(foreach PKG,$(SUBPKGS),$(GOLINT) -set_exit_status $(PKG)/...;)
# 	@$(GOLINT) -set_exit_status .
