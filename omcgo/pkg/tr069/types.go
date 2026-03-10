package tr069

import "time"

// DeviceId uniquely identifies a CPE device per TR069 spec.
type DeviceId struct {
	Manufacturer string `xml:"Manufacturer" json:"manufacturer"`
	OUI          string `xml:"OUI" json:"oui"`
	ProductClass string `xml:"ProductClass" json:"product_class"`
	SerialNumber string `xml:"SerialNumber" json:"serial_number"`
}

// ParameterValueStruct represents a single parameter name-value pair.
type ParameterValueStruct struct {
	Name  string `xml:"Name" json:"name"`
	Value string `xml:"Value" json:"value"`
	Type  string `xml:"type,attr,omitempty" json:"type,omitempty"`
}

// EventStruct represents a single event in an Inform message.
type EventStruct struct {
	EventCode  string `xml:"EventCode" json:"event_code"`
	CommandKey string `xml:"CommandKey" json:"command_key"`
}

// InformMessage is the CPE → ACS Inform request.
type InformMessage struct {
	DeviceId      DeviceId               `xml:"DeviceId" json:"device_id"`
	Event         []EventStruct          `xml:"Event>EventStruct" json:"event"`
	MaxEnvelopes  int                    `xml:"MaxEnvelopes" json:"max_envelopes"`
	CurrentTime   time.Time              `xml:"CurrentTime" json:"current_time"`
	RetryCount    int                    `xml:"RetryCount" json:"retry_count"`
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct" json:"parameter_list"`
}

// InformResponse is the ACS → CPE Inform response.
type InformResponse struct {
	MaxEnvelopes int `xml:"MaxEnvelopes" json:"max_envelopes"`
}

// --- RPC Request/Response Types ---

// GetParameterValuesRequest requests parameter values from the CPE.
type GetParameterValuesRequest struct {
	ParameterNames []string `xml:"ParameterNames>string" json:"parameter_names"`
}

// GetParameterValuesResponse contains parameter values from the CPE.
type GetParameterValuesResponse struct {
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct" json:"parameter_list"`
}

// SetParameterValuesRequest sets parameter values on the CPE.
type SetParameterValuesRequest struct {
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct" json:"parameter_list"`
	ParameterKey  string                 `xml:"ParameterKey" json:"parameter_key"`
}

// SetParameterValuesResponse is the CPE's response to SetParameterValues.
type SetParameterValuesResponse struct {
	Status int `xml:"Status" json:"status"` // 0=immediate, 1=reboot required
}

// GetParameterNamesRequest discovers parameter paths on the CPE.
type GetParameterNamesRequest struct {
	ParameterPath string `xml:"ParameterPath" json:"parameter_path"`
	NextLevel     bool   `xml:"NextLevel" json:"next_level"`
}

// ParameterInfoStruct describes a parameter's path and writability.
type ParameterInfoStruct struct {
	Name     string `xml:"Name" json:"name"`
	Writable bool   `xml:"Writable" json:"writable"`
}

// GetParameterNamesResponse is the CPE's response to GetParameterNames.
type GetParameterNamesResponse struct {
	ParameterList []ParameterInfoStruct `xml:"ParameterList>ParameterInfoStruct" json:"parameter_list"`
}

// AddObjectRequest creates a new multi-instance object on the CPE.
type AddObjectRequest struct {
	ObjectName   string `xml:"ObjectName" json:"object_name"`
	ParameterKey string `xml:"ParameterKey" json:"parameter_key"`
}

// AddObjectResponse is the CPE's response to AddObject.
type AddObjectResponse struct {
	InstanceNumber int `xml:"InstanceNumber" json:"instance_number"`
	Status         int `xml:"Status" json:"status"`
}

// DeleteObjectRequest removes a multi-instance object from the CPE.
type DeleteObjectRequest struct {
	ObjectName   string `xml:"ObjectName" json:"object_name"`
	ParameterKey string `xml:"ParameterKey" json:"parameter_key"`
}

// DeleteObjectResponse is the CPE's response to DeleteObject.
type DeleteObjectResponse struct {
	Status int `xml:"Status" json:"status"`
}

// DownloadRequest instructs the CPE to download a file.
type DownloadRequest struct {
	CommandKey     string `xml:"CommandKey" json:"command_key"`
	FileType       string `xml:"FileType" json:"file_type"`
	URL            string `xml:"URL" json:"url"`
	Username       string `xml:"Username" json:"username"`
	Password       string `xml:"Password" json:"password"`
	FileSize       int64  `xml:"FileSize" json:"file_size"`
	TargetFileName string `xml:"TargetFileName" json:"target_file_name"`
	DelaySeconds   int    `xml:"DelaySeconds" json:"delay_seconds"`
	SuccessURL     string `xml:"SuccessURL" json:"success_url"`
	FailureURL     string `xml:"FailureURL" json:"failure_url"`
}

// DownloadResponse is the CPE's response to Download.
type DownloadResponse struct {
	Status       int       `xml:"Status" json:"status"` // 0=complete, 1=in progress
	StartTime    time.Time `xml:"StartTime" json:"start_time"`
	CompleteTime time.Time `xml:"CompleteTime" json:"complete_time"`
}

// UploadRequest instructs the CPE to upload a file.
type UploadRequest struct {
	CommandKey   string `xml:"CommandKey" json:"command_key"`
	FileType     string `xml:"FileType" json:"file_type"`
	URL          string `xml:"URL" json:"url"`
	Username     string `xml:"Username" json:"username"`
	Password     string `xml:"Password" json:"password"`
	DelaySeconds int    `xml:"DelaySeconds" json:"delay_seconds"`
}

// UploadResponse is the CPE's response to Upload.
type UploadResponse struct {
	Status       int       `xml:"Status" json:"status"`
	StartTime    time.Time `xml:"StartTime" json:"start_time"`
	CompleteTime time.Time `xml:"CompleteTime" json:"complete_time"`
}

// RebootRequest instructs the CPE to reboot.
type RebootRequest struct {
	CommandKey string `xml:"CommandKey" json:"command_key"`
}

// RebootResponse is the CPE's response to Reboot.
type RebootResponse struct{}

// FactoryResetResponse is the CPE's response to FactoryReset.
type FactoryResetResponse struct{}

// ScheduleInformRequest asks the CPE to send an Inform after a delay.
type ScheduleInformRequest struct {
	DelaySeconds int    `xml:"DelaySeconds" json:"delay_seconds"`
	CommandKey   string `xml:"CommandKey" json:"command_key"`
}

// ScheduleInformResponse is the CPE's response to ScheduleInform.
type ScheduleInformResponse struct{}

// TransferComplete is sent by the CPE after a file transfer completes.
type TransferComplete struct {
	CommandKey   string    `xml:"CommandKey" json:"command_key"`
	FaultStruct  *Fault    `xml:"FaultStruct" json:"fault_struct,omitempty"`
	StartTime    time.Time `xml:"StartTime" json:"start_time"`
	CompleteTime time.Time `xml:"CompleteTime" json:"complete_time"`
}

// AutonomousTransferComplete is sent by the CPE after an autonomous
// (CPE-initiated) file transfer completes (e.g., PM/MR file upload).
type AutonomousTransferComplete struct {
	AnnounceURL    string    `xml:"AnnounceURL" json:"announce_url"`
	TransferURL    string    `xml:"TransferURL" json:"transfer_url"`
	IsDownload     bool      `xml:"IsDownload" json:"is_download"`
	FileType       string    `xml:"FileType" json:"file_type"`
	FileSize       int64     `xml:"FileSize" json:"file_size"`
	TargetFileName string    `xml:"TargetFileName" json:"target_file_name"`
	FaultStruct    *Fault    `xml:"FaultStruct" json:"fault_struct,omitempty"`
	StartTime      time.Time `xml:"StartTime" json:"start_time"`
	CompleteTime   time.Time `xml:"CompleteTime" json:"complete_time"`
}

// --- Parameter Attributes Types ---

// SetParameterAttributeStruct defines notification and access control
// settings for a single parameter (used in SetParameterAttributes request).
type SetParameterAttributeStruct struct {
	Name               string   `xml:"Name" json:"name"`
	NotificationChange bool     `xml:"NotificationChange" json:"notification_change"`
	Notification       int      `xml:"Notification" json:"notification"` // 0=off, 1=passive, 2=active
	AccessListChange   bool     `xml:"AccessListChange" json:"access_list_change"`
	AccessList         []string `xml:"AccessList>string" json:"access_list"`
}

// ParameterAttributeStruct describes the notification and access settings
// of a parameter (returned by GetParameterAttributes response).
type ParameterAttributeStruct struct {
	Name         string   `xml:"Name" json:"name"`
	Notification int      `xml:"Notification" json:"notification"`
	AccessList   []string `xml:"AccessList>string" json:"access_list"`
}

// GetParameterAttributesResponse is the CPE's response to GetParameterAttributes.
type GetParameterAttributesResponse struct {
	ParameterList []ParameterAttributeStruct `xml:"ParameterList>ParameterAttributeStruct" json:"parameter_list"`
}

// SetParameterAttributesResponse is the CPE's response to SetParameterAttributes.
type SetParameterAttributesResponse struct{}
