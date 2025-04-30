
NUM_LINKED_GOMODS=`cat go.mod | grep -v "^\/\/" | grep replace | wc -l | sed -e "s/ *//g"`



run: build
	cp .env /tmp 
	cp .env.dev /tmp
	LEETCOACH_ENV=dev LEETCOACH_WEB_PORT=:8080 air

checklinks:
	@if [ x"${NUM_LINKED_GOMODS}" != "x0" ]; then	\
		echo "You are trying to deploying with symlinks.  Remove them first and make sure versions exist" && false ;	\
	fi

deploy: checklinks build
	gcloud app deploy --project leetcoach --verbosity=info

build: webbuild resymlink

cligen:
	npx @openapitools/openapi-generator-cli generate 		\
  -i gen/openapiv2/services.swagger.json 							\
  -g typescript-fetch                                 \
  -o ./web/views/components/apiclient 								\
  --additional-properties=supportsES6=true,typescriptThreePlus=true,useSingleRequestParameter=true # Common useful options

webbuild:
	cd web ; npm run build

resymlink:
	mkdir -p locallinks
	rm -Rf locallinks/*
	cd locallinks && ln -s ../../templar
	cd locallinks && ln -s ../../s3gen
	cd locallinks && ln -s ../../goutils
	cd locallinks && ln -s ../../oneauth

prompt:
	source ~/personal/.shhelpers && files_for_llm `find . | grep -v sdl | grep -v apiclient | grep -v pnpm | grep -v llmprompts | grep -v node.mod | grep -v .git | grep -v sdl | grep -v bot | grep -v gen | grep -v web.static | grep -v output | grep -v content | grep -v dist | grep -v web.compo | grep -v web.templ | grep -e "\.go" -e "\.ts" -e "\.tsx" -e "\.html" -e "\.md"`
