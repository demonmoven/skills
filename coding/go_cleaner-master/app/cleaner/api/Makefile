gen_web:
	cd web && pwd && npm run build
	rm -rf ./static
	mkdir ./static
	cp -r ./web/build/* ./static
	git add ./static