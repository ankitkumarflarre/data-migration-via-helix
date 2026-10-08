# Rater → Helix migrator. `make run` builds the UI and the server and starts it.
ENV_FILE ?= .env
LEDGER   ?= data/ledger.jsonl
ADDR     ?= 127.0.0.1:8080
REPORT   ?= reference/Manatee FL Select HO Rater Effective 12.1.25 - Schema Validation Report.html
DDL      ?= ../pythonProjects/ddl.sql
RULESET  ?= manatee_fl_select_ho_12_1_25

.PHONY: all ui build run quote-run test check rules cleanup clean

all: build

ui:
	cd web && npm ci --silent && npm run build

build: ui
	go build -o bin/migrator ./cmd/migrator

run: build
	./bin/migrator serve -addr $(ADDR) -env-file $(ENV_FILE) -ledger $(LEDGER)

quote-run: build
	./bin/migrator serve -offline -addr $(ADDR) -quote-dir data/quotes

test:
	go vet ./...
	go test ./... -count=1

check: test
	cd web && npm run check && npm test

# Regenerate the committed rule set from the mapping report (review the diff!).
rules:
	go run ./cmd/rulegen -report "$(REPORT)" -ddl $(DDL) \
	  -pins internal/rules/rulesets/$(RULESET).pins.json -out internal/rules/rulesets/$(RULESET).json

# Delete from Helix every record the migrator created (asks for -yes).
cleanup:
	go run ./cmd/migrator cleanup -env-file $(ENV_FILE) -ledger $(LEDGER)

clean:
	rm -rf bin web/dist/assets
