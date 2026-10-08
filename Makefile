# Change to your Pi's user@host
PI ?= fred@raspberrypi.local

.PHONY: setup dev-backend dev-frontend build deploy logs message

setup:
	cd backend && go mod tidy
	cd frontend && npm install
	@test -f backend/config.json || cp backend/config.example.json backend/config.json

# Run these two in separate terminals, then open http://localhost:5173
dev-backend:
	cd backend && go run . -config config.json

dev-frontend:
	cd frontend && npm run dev

build:
	cd frontend && npm run build
	mkdir -p dist
	cd backend && GOOS=linux GOARCH=arm64 go build -o ../dist/mirror-server .

deploy: build
	ssh $(PI) 'mkdir -p ~/mirror/web'
	rsync -az --delete frontend/dist/ $(PI):mirror/web/
	rsync -az dist/mirror-server backend/config.json deploy/setup-pi.sh $(PI):mirror/
	ssh $(PI) 'sudo systemctl restart mirror.service 2>/dev/null || echo "First deploy: run bash ~/mirror/setup-pi.sh on the Pi"'

logs:
	ssh $(PI) 'journalctl -u mirror.service -f'

# Push a test message:  make message TEXT="Hello from the Mac"
message:
	curl -s -X POST http://$(lastword $(subst @, ,$(PI))):8080/api/events \
	  -H 'Content-Type: application/json' \
	  -d '{"text":"$(TEXT)","source":"Test"}'
