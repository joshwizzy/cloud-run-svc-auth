In this tutorial, we will demonstrate how to set up service-to-service authentication for Google Cloud Run services using the Go programming language. 

We will cover the following steps:

1. Deploy two services - a sender and a receiver. Initially, the receiver will not require any authentication. 
2. Add authentication to the receiver and demonstrate that the sender's requests will fail. 
3. Authorize the sender identity to the receive and demonstrate that the sender's requests will  succeed

## Prerequisites

Before you can begin this tutorial, you will need the following:

1. A Google Cloud Platform (GCP) account and a new project
2. Enable billing for the new project
3. Install and configure the Google Cloud SDK (gcloud) on your local machine
4. Enable the required APIs

### Creating a new project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Click the project drop-down and select or create the project you want to use for this tutorial.
3. Take note of your project ID, as you will need it later.

### Enable billing

1. In the [Cloud Console](https://console.cloud.google.com/), open the main menu and select "Billing".
2. Click "Link a billing account" and follow the steps to set up billing for your project.

### Install and configure the Google Cloud SDK (gcloud)

1. Download and install the [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) for your operating system.
2. Authenticate to your Google Cloud account using the gcloud tool:

```sh
$ gcloud auth login
```

This command will open a new browser window, asking you to log in with your Google account and grant permission to access your GCP resources.

1. Set up the default project and region:

```sh
$ gcloud config set project <YOUR_PROJECT_ID>
$ gcloud config set compute/region <YOUR_REGION>
```

Accept to enable the `compute.googleapis.com` API on the project if prompted.

Replace `<YOUR_PROJECT_ID>` with the project ID you noted earlier and `<YOUR_REGION>` with a desired region, such as `us-central1`.

Set the `PROJECT_ID` and `REGION` variables using the currently configured project and region:

```sh
$ PROJECT_ID=$(gcloud config get-value project)
$ REGION=$(gcloud config get-value compute/region)
```

### Enable the required APIs

1. In the [Cloud Console](https://console.cloud.google.com/), open the main menu and select "APIs & Services" > "Library".
2. Search for "Cloud Run" and click on "Cloud Run API".
3. Click "Enable" to enable the Cloud Run API for your project.
4. Repeat the process for "Cloud Build API" and "Identity Token Service API".

You can also enable the required APIs using the `gcloud` command-line tool by running the following commands:

Enable the Cloud Run API:

```sh 
$ gcloud services enable run.googleapis.com
```

Enable the Cloud Build API:

```sh
$ gcloud services enable cloudbuild.googleapis.com
```

Enable the Identity Token Service API (also known as the IAM Service Account Credentials API):

```sh
$ gcloud services enable iamcredentials.googleapis.com
```

These commands will enable the respective APIs for your current GCP project. 

Each of the three APIs serves a specific purpose, and enabling them ensures that the required services are accessible in your project.

1. **Cloud Run API**: The Cloud Run API is necessary for deploying, managing, and scaling your containerized applications on Google Cloud Run. 
2. **Cloud Build API**: The Cloud Build API allows you to build, test, and deploy your source code on Google Cloud. In this tutorial, you use the `gcloud builds submit` command to build your Docker images and push them to Google Container Registry.
3. **Identity Token Service API** (IAM Service Account Credentials API): This API enables you to generate access tokens, sign JSON Web Tokens (JWTs), and create OIDC ID tokens for service accounts. In the tutorial, you use service-to-service authentication, which requires generating identity tokens for your services. 

Now you're ready to proceed with the tutorial on service-to-service authentication for Google Cloud Run using Go.

## Deploy sending and receiving services

Create a directory for the project source code

`$ mkdir ${PROJECT_ID} && cd ${PROJECT_ID}`

### Receiving Service

First, let's create the receiving service. Create a new directory named `receiving-service` and navigate to it:

```sh
$ mkdir receiving-service && cd receiving-service && go mod init receiver
```

Create a file named `main.go` and add the following code:

```go
package main

import (
  "fmt"
  "net/http"
)

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello from the receiving service!")
  })

  http.ListenAndServe(":8080", nil)
}
```

Create a `Dockerfile` in the same directory with the following content:

```dockerfile
# Build stage
FROM golang:1.17 AS builder

# Set the working directory
WORKDIR /app

# Copy Go module files
COPY go.* ./

# Download dependencies
RUN go mod download

# Copy Go files
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Final stage
FROM alpine:3.15

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server /app/server

# Expose the listening port
EXPOSE 8080

# Run the server
CMD ["/app/server"]
```

Now, build and push the Docker image to Google Container Registry (GCR):

```sh
$ gcloud builds submit --tag gcr.io/${PROJECT_ID}/receiving-service
```

After the build is complete, deploy the service to Cloud Run:

```sh
$ gcloud run deploy receiving-service --image gcr.io/${PROJECT_ID}/receiving-service --region ${REGION} --platform managed --allow-unauthenticated
```

Set the `RECEIVING_SERVICE_URL` variable to the value of the receiving service URL after deployment.

```sh
$ RECEIVING_SERVICE_URL=$(gcloud run services describe receiving-service --region $REGION --format='value(status.url)')
```

Make a request to the receiving service URL and confirm that it returns the expected response

```sh
$ curl ${RECEIVING_SERVICE_URL}
Hello from the receiving service!                
```



### Sending Service

Create a new directory named `sending-service` and navigate to it:

```sh
$ cd ../ && mkdir sending-service && cd sending-service && go mod init sender
```

Create a file named `main.go` and add the following code:

```go
package main

import (
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
)

func main() {
  receivingServiceURL := os.Getenv("RECEIVING_SERVICE_URL")
  if receivingServiceURL == "" {
    log.Fatal("RECEIVING_SERVICE_URL environment variable is not set")
  }

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    resp, err := http.Get(receivingServiceURL)
    if err != nil {
      log.Printf("Failed to make request: %v", err)
      http.Error(w, "Failed to make request", http.StatusInternalServerError)
      return
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      log.Printf("Failed to read response body: %v", err)
      http.Error(w, "Failed to read response body", http.StatusInternalServerError)
      return
    }

    fmt.Fprintf(w, "Response from receiving service: %s", string(body))
  })

  http.ListenAndServe(":8080", nil)
}
```

Create a `Dockerfile` in the same directory with the following content:

```dockerfile
# Build stage
FROM golang:1.17 AS builder

# Set the working directory
WORKDIR /app

# Copy Go module files
COPY go.* ./

# Download dependencies
RUN go mod download

# Copy Go files
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Final stage
FROM alpine:3.15

# Install CA certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server /app/server

# Expose the listening port
EXPOSE 8080

# Run the server
CMD ["/app/server"]
```

Now, build and push the Docker image to Google Container Registry (GCR):

```sh
$ gcloud builds submit --tag gcr.io/${PROJECT_ID}/sending-service
```

After the build is complete, deploy the service to Cloud Run:

```sh
$ gcloud run deploy sending-service --image gcr.io/${PROJECT_ID}/sending-service --region ${REGION} --platform managed --allow-unauthenticated --set-env-vars RECEIVING_SERVICE_URL=${RECEIVING_SERVICE_URL}
```

Set the `SENDING_SERVICE_URL` variable to the value of the sending service URL after deployment.

```sh
$ SENDING_SERVICE_URL=$(gcloud run services describe sending-service --region $REGION --format='value(status.url)')`
```

Make a request to the receiving service URL and confirm that it returns the expected response

```sh
$ curl ${RECEIVING_SERVICE_URL}
Hello from the receiving service!                
```

Visit the sending service URL in your browser, and you should see the message from the receiving service, indicating that the call to the receiving service succeeded.

```sh
$ curl ${SENDING_SERVICE_URL}
Response from receiving service: Hello from the receiving service!
```



### Update the receiving service

First, set the `--no-allow-unauthenticated` flag while deploying the receiving service to require authentication:

```sh
$ gcloud run deploy receiving-service --image gcr.io/${PROJECT_ID}/receiving-service --region ${REGION} --platform managed --no-allow-unauthenticated
```

Visit the sending service URL in your browser, and you should see an error message, indicating that the call to the receiving service failed.

```sh
$ curl ${SENDING_SERVICE_URL}
Response from receiving service: 
<html><head>
<meta http-equiv="content-type" content="text/html;charset=utf-8">
<title>403 Forbidden</title>
</head>
<body text=#000000 bgcolor=#ffffff>
<h1>Error: Forbidden</h1>
<h2>Your client does not have permission to get URL <code>/</code> from this server.</h2>
<h2></h2>
</body></html>

```



### Update the sending service

In the `sending-service/main.go` file, add import the required packages:

```go
import (
  ...
  "context"
  "time"
  "google.golang.org/api/idtoken"
)
```

Then create a new function `httpClientWithIDToken` to generate an authenticated HTTP client:

```go
func httpClientWithIDToken(ctx context.Context, audience string) (*http.Client, error) {
  client, err := idtoken.NewClient(ctx, audience)
  if err != nil {
    return nil, err
  }
  return client, nil
}
```

Run `go mod tidy` to download the required package.

Modify the `main` function to use the authenticated HTTP client:

```go
func main() {
  receivingServiceURL := os.Getenv("RECEIVING_SERVICE_URL")
  if receivingServiceURL == "" {
    log.Fatal("RECEIVING_SERVICE_URL environment variable is not set")
  }

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    client, err := httpClientWithIDToken(ctx, receivingServiceURL)
    if err != nil {
      log.Printf("Failed to create authenticated client: %v", err)
      http.Error(w, "Failed to create authenticated client", http.StatusInternalServerError)
      return
    }

    resp, err := client.Get(receivingServiceURL)
    if err != nil {
      log.Printf("Failed to make request: %v", err)
      http.Error(w, "Failed to make request", http.StatusInternalServerError)
      return
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      log.Printf("Failed to read response body: %v", err)
      http.Error(w, "Failed to read response body", http.StatusInternalServerError)
      return
    }

    fmt.Fprintf(w, "Response from receiving service: %s", string(body))
  })

  http.ListenAndServe(":8080", nil)
}
```

## Rebuild and redeploy the sending service:

```sh
$ gcloud builds submit --tag gcr.io/${PROJECT_ID}/sending-service`
```

Now, deploy the sending service

```sh
$ gcloud run deploy sending-service --image gcr.io/${PROJECT_ID}/sending-service --region ${REGION} --platform managed --allow-unauthenticated --set-env-vars RECEIVING_SERVICE_URL=${RECEIVING_SERVICE_URL}

```

## Test the authentication

Visit the sending service URL in your browser, and you should now see a successful response from the receiving service

```sh
curl ${SENDING_SERVICE_URL}                                                                                                                                                                               
Response from receiving service: Hello from the receiving service!
```

We have identified our calling service to the receiving service, but we haven't set up the receiving service to accept requests from the calling service.

By default, Cloud Run services utilize the Compute Engine default service account, which has the Project > Editor IAM role. This grants your Cloud Run revisions read and write access to all resources in your GCP project.

To adhere to the principle of least privilege, we will create a new service account that has limited permissions.

### Create a new service account for the calling service

Run the following command to create a new service account:

```sh
$ gcloud iam service-accounts create calling-service-sa --display-name "Calling Service Account"
```

### Redeploy the calling service with the new service account

Redeploy the calling service using the new service account:

```sh
$ gcloud run deploy sending-service --image gcr.io/${PROJECT_ID}/sending-service --region ${REGION} --platform managed --allow-unauthenticated --service-account calling-service-sa@${PROJECT_ID}.iam.gserviceaccount.com --set-env-vars RECEIVING_SERVICE_URL=${RECEIVING_SERVICE_URL}
```

Now, if you visit the sending service URL in your browser, you should see a failed response because the calling service doesn't have permission to invoke the receiving service.

```go
$ curl ${SENDING_SERVICE_URL}                                                                   
Response from receiving service: 
<html><head>
<meta http-equiv="content-type" content="text/html;charset=utf-8">
<title>403 Forbidden</title>
</head>
<body text=#000000 bgcolor=#ffffff>
<h1>Error: Forbidden</h1>
<h2>Your client does not have permission to get URL <code>/</code> from this server.</h2>
<h2></h2>
</body></html>

```



### Grant the calling service identity permission to invoke the receiving service

To grant the calling service account permission to invoke the receiving service, run the following command:

```sh
$ gcloud run services add-iam-policy-binding receiving-service --region ${REGION} --member=serviceAccount:calling-service-sa@${PROJECT_ID}.iam.gserviceaccount.com --role=roles/run.invoker
```

### Step 4: Test the authentication

Visit the sending service URL in your browser, and you should now see a successful response from the receiving service, authenticated using the new service account with limited permissions. 

```sh
curl ${SENDING_SERVICE_URL}
Response from receiving service: Hello from the receiving service!
```



## Conclusion

In this tutorial, we demonstrated how to set up service-to-service authentication for Google Cloud Run services using the Go programming language.