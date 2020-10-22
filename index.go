package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func main() {
	var clusterName string
	flag.StringVar(&clusterName, "c", "", "Cluster name.")

	var serviceName string
	flag.StringVar(&serviceName, "s", "", "Service name")

	var taskFamily string
	flag.StringVar(&taskFamily, "t", "", "Task family name")

	var imageVersions string
	flag.StringVar(&imageVersions, "v", "", "New container versions")

	var region string
	flag.StringVar(&region, "r", "", "Region")

	flag.Parse()

	if clusterName == "" {
		fmt.Println("exit: No CLUSTER NAME specified, please use option: -c Example: -c ctrade-TEST-cluster")
		return
	}
	if serviceName == "" {
		fmt.Println("exit: No SERVICE NAME specified, please use option: -s Example: -s bff")
		return
	}
	if taskFamily == "" {
		fmt.Println("exit: No TASK FAMILY specified, please use option: -t Example: -t bff-TEST")
		return
	}
	if imageVersions == "" {
		fmt.Println("exit: No IMAGE VERSION specified, please use option: -v Example: -v registry.gitlab.com/modulus-derivatives/goesoteric/bff:20201012.1")
		return
	}
	if region == "" {
		fmt.Println("exit: No REGION specified, please use option: -r Example: -r us-west-2")
		return
	}

	fmt.Println("Cluster name: ", clusterName)
	fmt.Println("Service name: ", serviceName)
	fmt.Println("Task family name: ", taskFamily)
	fmt.Println("New container image version: ", imageVersions)
	fmt.Println("Region: ", region)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	// Create ECS service client
	svc := ecs.New(sess)

	describeTaskDefinitionInput := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskFamily),
	}

	// Get the latest version of the task definition
	latestTaskDef, err := svc.DescribeTaskDefinition(describeTaskDefinitionInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	latestContainerDef := latestTaskDef.TaskDefinition.ContainerDefinitions

	inputImageVersions := strings.Split(imageVersions, ",")

	// Start replacing the old version with the new one if matching image name are found
	for index := 0; index < len(inputImageVersions); index++ {
		// Look for matching image name and replace its current version with the new one
		currentInputNewImageVersion := inputImageVersions[index]
		// log.Println("Current image: ", currentInputNewImageVersion)
		currentInputNewImageVersionSplit := strings.Split(currentInputNewImageVersion, ":")
		if len(currentInputNewImageVersionSplit) != 2 {
			fmt.Println("exit: value of -v has to be like <image path>:<image version>")
			return
		}
		for count := 0; count < len(latestContainerDef); count++ {
			// log.Println("Current image: ", string(currentInputNewImageVersion))
			if strings.Contains(*latestContainerDef[count].Image, currentInputNewImageVersionSplit[0]) {
				latestContainerDef[count].Image = &inputImageVersions[index]
			}
		}
	}

	taskDefinitionInput := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    latestContainerDef,
		Family:                  aws.String(taskFamily),
		ExecutionRoleArn:        aws.String(*latestTaskDef.TaskDefinition.ExecutionRoleArn),
		TaskRoleArn:             aws.String(*latestTaskDef.TaskDefinition.TaskRoleArn),
		NetworkMode:             aws.String(*latestTaskDef.TaskDefinition.NetworkMode),
		Volumes:                 latestTaskDef.TaskDefinition.Volumes,
		PlacementConstraints:    latestTaskDef.TaskDefinition.PlacementConstraints,
		RequiresCompatibilities: latestTaskDef.TaskDefinition.RequiresCompatibilities,
		Cpu:                     aws.String(*latestTaskDef.TaskDefinition.Cpu),
		Memory:                  aws.String(*latestTaskDef.TaskDefinition.Memory),
	}

	taskDefWithNewImage, err := svc.RegisterTaskDefinition(taskDefinitionInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	taskDefRevision := strconv.FormatInt(*taskDefWithNewImage.TaskDefinition.Revision, 10)
	fmt.Println("Task Definition registered: " + taskFamily + ":" + taskDefRevision)

	updateServiceInput := &ecs.UpdateServiceInput{
		Cluster:        aws.String(clusterName),
		Service:        aws.String(serviceName),
		TaskDefinition: aws.String(taskFamily + ":" + taskDefRevision),
	}

	currentTime := time.Now()
	withNanoseconds := currentTime.String()
	fmt.Printf("Started updating service(%s): %s\n", serviceName, string(withNanoseconds))
	updateServiceResult, err := svc.UpdateService(updateServiceInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			case ecs.ErrCodeServiceNotFoundException:
				fmt.Println(ecs.ErrCodeServiceNotFoundException, aerr.Error())
			case ecs.ErrCodeServiceNotActiveException:
				fmt.Println(ecs.ErrCodeServiceNotActiveException, aerr.Error())
			case ecs.ErrCodePlatformUnknownException:
				fmt.Println(ecs.ErrCodePlatformUnknownException, aerr.Error())
			case ecs.ErrCodePlatformTaskDefinitionIncompatibilityException:
				fmt.Println(ecs.ErrCodePlatformTaskDefinitionIncompatibilityException, aerr.Error())
			case ecs.ErrCodeAccessDeniedException:
				fmt.Println(ecs.ErrCodeAccessDeniedException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println("Update service result:")
	fmt.Println(updateServiceResult)

	describeServicesInput := &ecs.DescribeServicesInput{
		Cluster: aws.String(clusterName),
		Services: []*string{
			aws.String(serviceName),
		},
	}

	waitErr := svc.WaitUntilServicesStable(describeServicesInput)
	if waitErr != nil {
		fmt.Println(waitErr.Error())
		return
	}

	currentTime = time.Now()
	withNanoseconds = currentTime.String()
	fmt.Printf("Done: %s\n", string(withNanoseconds))

}
