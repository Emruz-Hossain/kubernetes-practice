This is practice project using client-go. Here deployment and service is created using client-go.

# Build Project
 ```
 go build -o cgp main.go
```
# Create deployment
```
./cgp createDeployment
```
This will create a deployment named ```wc-deployment``` with 2 replica which run a simple webcalculator. 
# Create Service
```
./cgp createService
```
This will create service named ```wc-service``` and show URL to access application using this service.

# URL request format
```
http://192.168.0.100:30000?FirstOperand=324&SecondOperand=32
```
# Credentials
```
User Name: emruz
Password: 1234
```
