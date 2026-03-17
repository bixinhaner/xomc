<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
    .padding-left-px {
        padding-left: 60px;
    }
    .padding-left-px .el-form-item {
        display: inline-block;
        margin-right: 30px;
    }

    #device_policy_add_ctn .el-form-item__label {
        line-height: 25px;
    }
    .splite-line {
        opacity: 0.2;
        margin-bottom: 30px;
    }
    .margin-left-px {
        margin-left: 35px;
        margin-bottom: 15px;
    }
    .prefix-title {
        display: inline-block;
        height: 30px;
        line-height: 30px;
        font-weight: bold;
        font-size: 14px;
        padding: 0px 30px 0 0;
    }
    .radio-group {
        width: 150px;
        display: flex;
        flex-direction: column;
    }
    .radio-group label {
        margin-left: 30px;
        margin-top: 20px;
    }
    .radio-group-title {
        padding: 10px 0 10px 15px;
        background-color: #e9e9e9;
    }
    .select-80px .el-input__inner,
    .select-height-26  .el-input__inner{
        max-height: 26px;
        height: 26px;
    }
    .select-80px .el-input {
        width: 80px;
    }

</style>

<div id="device_policy_add_ctn" style="height: 100%; display: flex; flex-direction: column;">
    <el-form ref="form" :model="form" :rules="rules" label-position="left" label-width="100" style="margin-top: 30px; flex: auto;overflow: auto;">
        <div class="group-title not-extend margin-left-px">
            <span class="title-icon"></span>
            <span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
        </div>
        <el-form-item label="<%=rb.getString("KPISheBei")%>" class="padding-left-px" prop="device">
            <el-checkbox-group v-model="form.device" @change="deviceChange">
                <el-checkbox label="enb">eNB</el-checkbox>
                <el-checkbox label="cpe">CPE</el-checkbox>
            </el-checkbox-group>
        </el-form-item>

        <div class="padding-left-px">
            <el-tabs v-model="activeName">
                <el-tab-pane v-if="form.device.includes('enb')" label="eNB" name="enb">
                    <el-pairgrid style="height: 300px;"
                        row-key="smallCellCode"
                        :left-url="enbURL"
                        :readonly="form.enbSelectAll=='true'"
                        :query-params="enbParams"
                        @selection-change="enbChange">
                        <template slot="prev">
                            <div style="border-left: 1px solid #e9e9e9;height: 100%;">
                                <div class="radio-group-title">Device Specific</div>

                                <el-radio-group class="radio-group" v-model="form.enbSelectAll" @change="enbGroupChange">
                                    <el-radio label="true">All</el-radio>
                                    <el-radio label="false">Select</el-radio>
                                </el-radio-group>
                            </div>
                        </template>
                        <template slot="toolbar">
                            <el-query @query="enbQuery" type="normal" placeholder="Serial Number / Cell Name"></el-query>
                        </template>
                        <template slot="left">
                            <el-table-column type="selection" :reserver-selection="true"></el-table-column>
	                        <el-table-column prop="connectionStatus" key="connectionStatus" width="40">
	                            <template slot-scope="scope">
	                                <div v-html="connStatusFmt(scope.row, scope.row['connectionStatus'], scope.$index)"></div>
	                            </template>
	                        </el-table-column>
                            <el-table-column label="Serial Number" prop="serialNumber"></el-table-column>
                            <el-table-column label="Cell Name" prop="cellName"></el-table-column>
                            <el-table-column label="Product Type" prop="productType"></el-table-column>
                            <el-table-column label="Device Group" prop="deviceGroup"></el-table-column>
                        </template>
                        <template slot="right">
                            <el-table-column label="Serial Number" prop="serialNumber"></el-table-column>
                            <el-table-column label="Device Group" prop="deviceGroup"></el-table-column>
                        </template>
                    </el-pairgrid>

                    <el-form-item prop="enbCode" label-width="0">
                        <el-input v-show="false" v-model="form.enbCode"></el-input>
                    </el-form-item>
                </el-tab-pane>
                <el-tab-pane v-if="form.device.includes('cpe')" label="CPE" name="cpe">
                    <el-pairgrid style="height: 300px;"
                        :left-url="cpeURL"
                        row-key="CPE_CODE"
                        :readonly="form.cpeSelectAll=='true'"
                        :query-params="cpeParams"
                        @selection-change="cpeChange">
                        <template slot="prev">
                            <div style="border-left: 1px solid #e9e9e9;height: 100%;">
                                <div class="radio-group-title">Device Specific</div>

                                <el-radio-group class="radio-group" v-model="form.cpeSelectAll" @change="cpeGroupChange">
                                    <el-radio label="true">All</el-radio>
                                    <el-radio label="false">Select</el-radio>
                                </el-radio-group>
                            </div>
                        </template>
                        <template slot="toolbar">
                            <el-query @query="cpeQuery" type="normal" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%>"></el-query>
                        </template>
                        <template slot="left">
                            <el-table-column type="selection" :reserver-selection="true"></el-table-column>
	                        <el-table-column prop="CONNECTION_STATUS" width="40">
	                            <template slot-scope="scope">
	                                <div v-html="cpeConnStatusFmt(scope.row, scope.row.CONNECTION_STATUS, scope.$index)"></div>
	                            </template>
	                        </el-table-column>
                            <el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER"></el-table-column>
                            <el-table-column label="<%=rb.getString("CPEName")%>" prop="CPE_NAME"></el-table-column>
                            <el-table-column label="MAC Address" prop="MACADDRESS"></el-table-column>
                            <el-table-column label="IMSI" prop="IMSI"></el-table-column>
                            <el-table-column label="<%=rb.getString("SheBeiZu")%>" prop="group_name"></el-table-column>
                        </template>
                        <template slot="right">
                            <el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER"></el-table-column>
                            <el-table-column label="<%=rb.getString("SheBeiZu")%>" prop="group_name"></el-table-column>
                        </template>
                    </el-pairgrid>

                    <el-form-item prop="cpeCode" label-width="0">
                        <el-input v-show="false" v-model="form.cpeCode"></el-input>
                    </el-form-item>
                </el-tab-pane>
            </el-tabs>
        </div>

        <div class="group-title not-extend margin-left-px" style="margin-top: 20px;">
            <span class="title-icon"></span>
            <span class="title-text">Select Policy</span>
        </div>
        <el-form-item label="APN Policy" class="padding-left-px" prop="apnPolicyName">
            <el-select v-model="form.apnPolicyName" @change="policyChange">
                <el-option v-for="item in policyNameList" :label="item" :value="item" :key="item"></el-option>
            </el-select>
        </el-form-item>

        <div v-show="form.apnPolicyName!=''" class="padding-left-px">
            <div style="border: 1px solid #E9E9E9;padding: 15px;border-right: none;">
                <div v-for="(row,idx) in apn.apnConfig" v-if="row.apnEnable == 'true'">
                    <el-checkbox v-model="row.apnEnable" style="margin-right: 20px;"  disabled
                        @change="enableChange"
                        true-label="true" false-label="false">
                    </el-checkbox>

                    <span class="prefix-title">APN{{idx+1}}</span>
                    <el-form-item label="APN Name" label-width="100" style="margin-right: 20px;">
                        <el-select v-model="row.apnName" class="select-height-26" disabled>
                            <el-option v-for="item in apnNameList" :label="item.label" :value="item.value"></el-option>
                        </el-select>
                    </el-form-item>

                    <el-form-item label="eNB Default APN" label-width="140" style="margin-right: 50px;">
                        <el-select v-model="row.defaultApn" class="select-80px" disabled>
                            <el-option label="True" value="1"></el-option>
                            <el-option label="False" value="0"></el-option>
                        </el-select>
                    </el-form-item>

                    <el-form-item label="Bearer Type" label-width="100" style="margin-right: 50px;">
                        <el-select v-model="row.bearType" class="select-80px" disabled>
                            <el-option label="MGMT" value="1"></el-option>
                            <el-option v-show="false" label="VOIP" value="2"></el-option>
                            <el-option label="DATA" value="0"></el-option>
                        </el-select>
                    </el-form-item>
                    
                    <el-form-item label="APN Type" label-width="100" style="margin-right: 50px;">
	                    <el-select v-model="row.greType" class="select-80px" disabled>
	                        <el-option label="Layer2" value="1"></el-option>
	                        <el-option label="Layer3" value="0"></el-option>
	                    </el-select>
	                </el-form-item>

                    <el-form-item :label="row.greType == '0'? 'VLAN ID':'VLAN List'" label-width="100" style="margin-right: 50px;">
                        <el-input v-model="row.vlanId" disabled></el-input>
                    </el-form-item>
                </div>
            </div>
            <div style="border: 1px solid #E9E9E9;padding: 15px 0;">
                <div class="group-title not-extend margin-left-px">
                    <span class="title-icon"></span>
                    <span class="title-text">eNB L2 Tunnel</span>
                </div>
                <div class="padding-left-px">
                    <el-form-item label="L2 Tunnel Enable" label-width="120">
                        <el-switch v-model="apn.enbL2TunnelEnable" disabled active-value="true" inactive-value="false"></el-switch>
                    </el-form-item>
                    <el-form-item label="L2 Server IP" label-width="120">
                        <el-input v-model="apn.enbL2ServerIp" disabled></el-input>
                    </el-form-item>
                    <el-form-item label="L2 Tunnel Mode" label-width="120">
                        <el-select v-model="apn.enbL2TunnelMode" disabled>
                            <el-option label="GRE" value="0"></el-option>
                            <el-option label="VxLan" value="1"></el-option>
                            <el-option label="GRE&VxLan" value="2"></el-option>
                        </el-select>
                    </el-form-item>
                </div>

                <hr class="splite-line">

                <div class="group-title not-extend margin-left-px">
                    <span class="title-icon"></span>
                    <span class="title-text">CPE L2 Tunnel</span>
                </div>
                <div class="padding-left-px">
                    <el-form-item label="Mode">
                        <el-select v-model="apn.cpeL2TunnelMode" disabled>
                            <el-option label="NAT" value="0"></el-option>
                            <el-option label="Router Mode" value="1"></el-option>
                            <el-option label="Tunnel Mode" value="2"></el-option>
                            <el-option label="Bridge Mode" value="3"></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="GRE Type">
                        <el-select v-model="apn.greType" disabled>
                            <el-option label="Layer2" value="1"></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="Destination IP">
                        <el-input v-model="apn.destinationIp" disabled></el-input>
                    </el-form-item>
                </div>
                
                <div class="group-title not-extend margin-left-px" style="padding-left: 25px;">
                    <span class="title-text">WAN Config</span>
                </div>
                <template v-for="(item, index) in apn.apnConfig">
                    <div class="padding-left-px" v-if="item.apnEnable == 'true'">
                        <span class="prefix-title">APN{{item.apnOrder}}</span>

                        <el-form-item label="Bearer Type">
                            <el-select v-model="item.bearType" disabled>
                                <el-option label="MGMT" value="1"></el-option>
                        		<el-option v-show="false" label="VOIP" value="2"></el-option>
                                <el-option label="DATA" value="0"></el-option>
                            </el-select>
                        </el-form-item>
                    </div>
                </template>
            </div>
        </div>
    </el-form>

    <div class="padding-left-px" style="padding-bottom: 20px;padding-top: 20px;">
        <el-button type="primary" @click="submit"><%=rb.getString("QueDing")%></el-button>
        <el-button @click="close"><%=rb.getString("QuXiao")%></el-button>
    </div>
</div>

<script>
    new Vue({
        el: '#device_policy_add_ctn',
        data() {
            var vm = this,
                validDevice = function(rule, val, cb) {
                    if(val && val.length > 0) {
                        cb();
                    }else {
                        cb('<%=rb.getString("SheBeiLeiXingWeiKongTiShi")%>');
                    }
                },
                validCode = function(rule, val, cb) {
                    var type = rule.flag,
                        allMap = {
                            cpe: 'cpeSelectAll',
                            enb: 'enbSelectAll'
                        },
                        codeMap = {
                            cpe: 'cpeCode',
                            enb: 'enbCode'
                        },
                        codeKey = codeMap[type],
                        allKey = allMap[type];

                    if(vm.form.device.includes(type) && vm.form[allKey] == 'false' && vm.form[codeKey]=='') {
                        cb('<%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%>');
                    }else {
                        cb()
                    }
                },
                validName = function(rule, val, cb) {
                    var reg = /^[a-zA-Z0-9]*$/;

                    if(reg.test(val)) {
                        cb();
                    }else {
                        cb('Character or number');
                    }
                };

            return {
                activeName: '',
                enbParams: {
                    searchText: ''
                },
                cpeParams: {
                    search_text: '',
                    like_fields: 'serial_number,CPE_NAME,IMSI,macaddress'
                },
                form: {
                    device: [],
                    apnPolicyName: '',
                    enbCode: '',
                    cpeCode: '',
                    enbSelectAll: 'false',
                    cpeSelectAll: 'false'
                },
                rules: {
                    device: [{validator: validDevice}],
                    apnPolicyName: [{validator: validName}],
                    enbCode: [{validator: validCode, flag: 'enb'}],
                    cpeCode: [{validator: validCode, flag: 'cpe'}]
                },
                apn: {
                    enbL2TunnelEnable: 'false',
                    enbL2ServerIp: '',
                    enbL2TunnelMode: '2',
                    cpeL2TunnelMode: '2',
                    destinationIp: '',
                    greType: '1',
                    apnConfig: [
                        {
                            apnOrder: 1,
                            apnEnable: 'false',
                            defaultApn: 'true',
                            bearType: '1',
                            apnName: '',
                            dhcpEnable: '1',
                            vpnType: '1',
                            greType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        },
                        {
                            apnOrder: 2,
                            apnEnable: 'false',
                            defaultApn: 'false',
                            bearType: '0',
                            apnName: '',
                            dhcpEnable: '1',
                            vpnType: '1',
                            greType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        },
                        {
                            apnOrder: 3,
                            apnEnable: 'false',
                            defaultApn: 'false',
                            bearType: '0',
                            apnName: '',
                            dhcpEnable: '1',
                            vpnType: '1',
                            greType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        },
                        {
                            apnOrder: 4,
                            apnEnable: 'false',
                            defaultApn: 'false',
                            bearType: '0',
                            apnName: '',
                            dhcpEnable: '1',
                            vpnType: '1',
                            greType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        }
                    ]
                },
                policyNameList: [],
                apnNameList: [],
                enbURL: '${ctx}/epc/apnpolicyapply/queryEnbPageList.action',
                cpeURL: '${ctx}/cell/CPE/queryCpeInfosList.action?type=0',
                cpeParams: {
                    TimeZone: timeZone,
                    isCloudCore: isCloudCore,
                    search_text: '',
                    like_fields: 'serial_number,CPE_NAME'
                }
            }
        },
        watch: {
            'form.device': function(val){
                var vm = this;

                if(val && val.length  == 1) {
                    vm.activeName = val[0];
                }
            }
        },
        methods: {
            init() {
                var vm = this;

                axios.post('${ctx}/epc/apnpolicyconfig/queryApnPolicyNameList.action').then(function(res){
                    var data = res.data||[];

                    vm.policyNameList = data;
                });
                
                axios.post('${ctx}/epc/apnconfig/queryApnNameList.action').then(function(res){
                    var data = res.data||[];

                    data.map(function(item) {
                        vm.apnNameList.push({
                            label: item,
                            value: item
                        });
                    });
                });
            },
            // table formatters
            connStatusFmt(row, value, index) {

                return connStatusFormatterSyn(value, row, index);
            },
            // table formatters
            cpeConnStatusFmt(row, value, index) {

            	return connStatusFormatter(value, row, index);
            },
            enbQuery(text) {
                this.enbParams.searchText = text;
            },
            cpeQuery(text) {
                this.cpeParams.search_text = text;
            },
            deviceChange(val) {
                var vm = this;

                if(val && !val.includes('enb')) {
                    vm.form.enbCode = '';
                    vm.form.enbSelectAll = 'false';
                }

                if(val && !val.includes('cpe')) {
                    vm.form.cpeCode = '';
                    vm.form.cpeSelectAll = 'false';
                }
            },
            enbChange(s) {
                var vm = this;

                vm.form.enbCode = s.map(function(row){
                    return row.smallCellCode;
                });
            },
            cpeChange(s) {
                var vm = this;

                vm.form.cpeCode = s.map(function(row){
                    return row.CPE_CODE;
                });
            },
            enbGroupChange(val) {
                var vm = this;

                if(val == 'all') {
                    //vm.form.enbCode = '';
                }
            },
            cpeGroupChange(val) {
                var vm = this;

                if(val == 'all') {
                    //vm.form.cpeCode = '';
                }
            },
            policyChange(val) {
                var vm = this,
                    params = {
                        apnPolicyName: val
                    };

                axios.post('${ctx}/epc/apnpolicyconfig/queryApnPolicyInfo.action', stringify(params)).then(function(res){
                    var data = res.data;

                    if(data) {
                        Object.assign(vm.apn, data);
                    }else {
                        Object.assign(vm.apn, {
                            enbL2TunnelEnable: 'false',
                            enbL2ServerIp: '',
                            enbL2TunnelMode: '2',
                            cpeL2TunnelMode: '2',
                            destinationIp: '',
                            apnConfig: [
                                {
                                    apnOrder: 1,
                                    apnEnable: 'false',
                                    defaultApn: '0',
                                    bearType: '1',
                                    apnName: '',
                                    vpnType: '1',
                                    greType: '1',
                                    tunnelIpAddress: '',
                                    tunnelSubnetMask: '',
                                    vlanId: ''
                                },
                                {
                                    apnOrder: 2,
                                    apnEnable: 'false',
                                    defaultApn: '0',
                                    bearType: '0',
                                    apnName: '',
                                    vpnType: '1',
                                    greType: '1',
                                    tunnelIpAddress: '',
                                    tunnelSubnetMask: '',
                                    vlanId: ''
                                },
                                {
                                    apnOrder: 3,
                                    apnEnable: 'false',
                                    defaultApn: '0',
                                    bearType: '0',
                                    apnName: '',
                                    vpnType: '1',
                                    greType: '1',
                                    tunnelIpAddress: '',
                                    tunnelSubnetMask: '',
                                    vlanId: ''
                                },
                                {
                                    apnOrder: 4,
                                    apnEnable: 'false',
                                    defaultApn: '0',
                                    bearType: '0',
                                    apnName: '',
                                    vpnType: '1',
                                    greType: '1',
                                    tunnelIpAddress: '',
                                    tunnelSubnetMask: '',
                                    vlanId: ''
                                }
                            ]
                        });
                    }
                });
            },
            enableChange(val) {
                var vm = this;
                
                vm.apn.apnConfig[0].apnDefault = vm.apn.apnConfig[0].bearType=='1'?'true':'false';
                vm.apn.apnConfig[1].apnDefault = vm.apn.apnConfig[1].bearType=='1'?'true':'false';
                vm.apn.apnConfig[2].apnDefault = vm.apn.apnConfig[2].bearType=='1'?'true':'false';
                vm.apn.apnConfig[3].apnDefault = vm.apn.apnConfig[3].bearType=='1'?'true':'false';
            },
            close() {
                eventBus.$emit('close-device-policy');
            },
            submit() {
                var vm = this,
                    params = {

                    };

                Object.assign(params, vm.form);
                delete params.device;

                if([true, 'true'].includes(vm.form.enbSelectAll)) {
                    vm.form.enbCode = '';
                }
                if([true, 'true'].includes(vm.form.cpeSelectAll)) {
                    vm.form.cpeCode = '';
                }

                vm.$refs.form.validate(function(r) {
                    if(r) {
                        axios.post('${ctx}/epc/apnpolicyapply/sendApnPolicy.action', stringify(params)).then(function(res){
                            var data = res.data;

                            if(data.success == true) {
                                vm.$message({
                                    message: '<%=rb.getString("ChengGongDouHaoChongQiShengXiao")%>',
                                    type: 'success'
                                });

                                vm.close();
                            }else {
                                vm.$message({
                                    message: data.message,
                                    type: 'error'
                                });
                            }
                        });
                    }
                });
            }
        },
        mounted() {
            this.init();
        }
    });
</script>
