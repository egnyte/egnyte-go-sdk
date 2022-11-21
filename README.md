Egnyte SDK
==========

This is the official Go client library for Egnyte's Public APIs.
For overview of the HTTP API, go to https://developers.egnyte.com

Getting an API key
==================

Register on https://developers.egnyte.com/member/register to get API key
for your Egnyte account. This key is required to generate an Egnyte OAuth
token.

Examples
========

* Include this library

```
    import "github.com/egnyte/egnyte-go-sdk/egnyte"
```    


* Generate an access token

```
    config := map[string]string{"api_key": "API_KEY", "<username>": "<UserName>", "password": "<PASSWORD>", "domain": "<DOMAIN>"}
    egnyte.GetAccessToken(context.Background(), Config)
```

* Create a client object

```
   client = egnyte.NewClient(context.Background(), "domain", "accessToken")
```  



* Create a folder

```
   folderObj := Object{
		Client:   client,
		Path:     <DestinationFolderPath>,
		IsFolder: true,
   }
   folder = folderObj.create(context.Background())
```


* Delete a folder
```
    folderObj := Object{
		Client:   client,
		Path:     <DestinationFolderPath>,
		IsFolder: true,
	}
    folderObj.delete()
```

* Get a list of files in a folder, download a file
```
    folder = client.object(<DestinationFolderPath>).list(ctx)
    for file_obj in folder.files:
        with file_obj.Get(context.Background()) as download:
            data = download.read()
      
```

* Get a list of files in a subfolders
```
    folder = client.object(<DestinationFolderPath>).list(ctx)
    for folder_obj in folder.folders:
        do_something(folder_obj)
        
```

* Upload a new file from local file

```
    in, err := os.OpenFile(<SourceFilePath>, os.O_RDWR, 0666)
	fileInfo, err := in.Stat()
	fileObj := Object{
		Client:  client,
		Path:    <DestinationFilePath>,
		Body:    in,
		Size:    int(fileInfo.Size()),
		ModTime: fileInfo.ModTime(),
	}
	fileObj.Create(context.Background())
```

* Delete a file

```
   fileObj.Delete(context.Background())
```

* Download a file
```
   fileObj.Get(context.Background())
```

* Get Event Cursor
````
   event = client.EventCursor(context.Background())
````


Full documentation
==================
Generate document locally

```
godoc -http=:6060 & open http://localhost:6060/pkg/github.com/egnyte/egnyte-go-sdk/egnyte
```

Command line
============

If you're using implicit flow, you'll need to provide access token directly.
If you're using API token with resource flow, you can generate API access token using command line options.
See the full documentation or build, then use:

```
   egnyte --help
```   


Create a build
==============

```
   go build -o egnyte 
```


Create configuration
====================

Configuration file will be created in config.json

```

   egnyte create_config -c [API_KEY] -d [DOMAIN] -p [PASSWORD] -u [USERNAME]
```



Running tests
=============

Tests can be run with directly on the egnyte package

```
   go test -v
```

    

In order to run tests, you need to create test configuration file: ~/.egnyte/test_config.json

```

    {
        "access_token": "access token you received after passing the auth flow", 
        "api_key": "key received after registering developer account",
        "domain": "Egnyte domain, e.g. example.egnyte.com",
        "username": "username of Egnyte admin user", 
        "password": "password of the same Egnyte admin user"
    }
```

Tests will be run against your domain on behalf on admin user.

Please refer to https://developers.egnyte.com/docs/read/Public_API_Authentication#Internal-Applications for information
about how to generate access token.

Helping with development
========================

If you'd like to fix something yourself, please fork this repository,
commit the fixes and updates to tests, then set up a pull request with
information what you're fixing.

Please remember to assign copyright of your fixes to Egnyte or make them
public domain so we can legally merge them.
