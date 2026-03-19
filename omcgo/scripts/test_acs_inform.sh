#!/bin/bash
# ACS Inform 测试脚本
# 使用方法: ./test_acs_inform.sh [ACS_URL]
# 示例: ./test_acs_inform.sh http://localhost:8080/acs
#       ./test_acs_inform.sh http://localhost:8080/acs bootstrap

ACS_URL="${1:-http://localhost:8080/acs}"
EVENT_TYPE="${2:-periodic}"

# 生成当前时间戳
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 生成随机 CWMP ID
CWMP_ID="cwmp-$(uuidgen 2>/dev/null || echo "test-$$-$(date +%s)")"

# 根据事件类型选择 EventCode
case "$EVENT_TYPE" in
  bootstrap|boot)
    EVENT_CODE="0 BOOTSTRAP"
    COMMAND_KEY="bootstrap-key-001"
    ;;
  periodic)
    EVENT_CODE="2 PERIODIC"
    COMMAND_KEY="periodic-key-001"
    ;;
  value)
    EVENT_CODE="4 VALUE CHANGE"
    COMMAND_KEY="value-change-001"
    ;;
  *)
    EVENT_CODE="2 PERIODIC"
    COMMAND_KEY="periodic-key-001"
    ;;
esac

echo "=== ACS Inform Test ==="
echo "URL: $ACS_URL"
echo "Event: $EVENT_CODE"
echo "CWMP ID: $CWMP_ID"
echo "========================"
echo ""

# 发送 Inform 请求
curl -v -X POST "$ACS_URL" \
  -H "Content-Type: text/xml; charset=utf-8" \
  -H "SOAPAction: " \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">'"$CWMP_ID"'</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>Baicells</Manufacturer>
        <OUI>001A2B</OUI>
        <ProductClass>SmallCell-LTE</ProductClass>
        <SerialNumber>BCTest00123456</SerialNumber>
      </DeviceId>
      <Event soap:arrayType="cwmp:EventStruct[1]">
        <EventStruct>
          <EventCode>'"$EVENT_CODE"'</EventCode>
          <CommandKey>'"$COMMAND_KEY"'</CommandKey>
        </EventStruct>
      </Event>
      <ParameterList soap:arrayType="cwmp:ParameterValueStruct[8]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.Manufacturer</Name>
          <Value xsi:type="xsd:string">Baicells</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.ProductClass</Name>
          <Value xsi:type="xsd:string">SmallCell-LTE</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SerialNumber</Name>
          <Value xsi:type="xsd:string">BCTest00123456</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.HardwareVersion</Name>
          <Value xsi:type="xsd:string">v2.0</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">1.5.3.2</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.ConnectionRequestURL</Name>
          <Value xsi:type="xsd:string">http://192.168.1.100:7547</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.PeriodicInformInterval</Name>
          <Value xsi:type="xsd:unsignedInt">300</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.Time.CurrentLocalTime</Name>
          <Value xsi:type="xsd:dateTime">'"$TIMESTAMP"'</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>'
