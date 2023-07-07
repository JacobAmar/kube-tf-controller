start: init deploy up

init:
	chmod +x ./registry.sh
	./registry.sh

deploy:
	kubectl apply -f ../crds/
	kubectl apply -f ../examples/

up:
	tilt up