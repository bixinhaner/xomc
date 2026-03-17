<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
    
    .task_count{
        position:absolute;
        right:30px;
        top:10px;
    }
    .task_count p,.device_count p{
        display:inline-block;
        margin-left:10px;
    }
    .task_count p span,.suc_count span,.fail_count span{
        height:28px;
        line-height:30px;
        padding: 0 10px;
        display:inline-block;
        font-size:12px;
        vertical-align:bottom;
    }

    .task_count p span:nth-child(odd){
        border-radius:4px 0px 0px 4px;
        border-right:0px;
    }
    .task_count p span:nth-child(even){
        border-radius:0px 4px 4px 0px;
        color:#333;
        font-weight:normal;
    }
    .task_count i,.suc_count i,.fail_count i{
        margin-right:5px;
        font-size:16px;
    }
    .suc_count,.fail_count{
        display:inline-block;
        margin-left:10px;
    }
    .suc_count span:first-child{
        border:1px solid #67D972;
        border-radius:4px 0px 0px 4px;
        border-right:0px;
        color:#67D972 !important;
        background:#EEFFF3;
    }
    .suc_count span:last-child{
        border:1px solid #67D972;
        border-radius:0px 4px 4px 0px;
        color:#333;
        font-weight:normal;
    }
    .fail_count span:first-child{
        border:1px solid #E88282;
        border-radius:4px 0px 0px 4px;
        border-right:0px;
        color:#E88282 !important;
        background:#FEF2F2;
    }
    .fail_count span:last-child{
        border:1px solid #E88282;
        border-radius:0px 4px 4px 0px;
        color:#333;
        font-weight:normal;
    }
    .suc_count .el-icon-circle-success:before{
        color:#67D972
    }
    .fail_count .el-icon-circle-close:before{
        color:#E88282
    }

    .apn-list {
        padding: 10px;
    }
    .apn-list div {
        margin-top: 5px;
    }
    .apn-title {
        font-weight: bold;
    }
    
    .active-status-div {
        display: flex;
        align-items: center
    }
    .active-status-div > i {
        margin-right: 5px;
        font-size: 20px;
    }
    .active-status-div .success-status::before {
        color: #67D972;
    }
</style>

<div id="result_device_ctn" style="height: 100%;display: flex;">
    <div style="position: absolute; top: 5px;z-index: 100;right: 25px;">
        <i v-if="writable" class="el-icon el-icon-circle-add" style="margin-right: 10px;" @click="addDevicePolicy"></i>
        <i class="el-icon el-icon-circle-close" style="margin-right: 10px;" @click="closeDevice"></i>
    </div>
    <el-tabs>
        <el-tab-pane label="eNB">
            <el-ctable ref="enb" id="device_enb_policy" :time="6"
                :url="enbURL"
                :query-params="enbParams"
                @load-success="loadSuccess">
                <template slot="toolbar">
                    <div style="display: flex; justify-content: space-between;">
                        <el-query type="normal" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>" @query="queryEnb"></el-query>
                        
                        <div class='device_count'>
                            <p class="suc_count"><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{resultEnb.success}}</span></p>
                            <p class="fail_count"><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{resultEnb.failure}}</span></p>
                        </div>
                    </div>
                </template>
                
                <el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serialNumber"></el-table-column>
                <el-table-column label="<%=rb.getString("HostName")%>" prop="cellName"></el-table-column>
                <el-table-column label="<%=rb.getString("ChangPinXingHao")%>" prop="productType"></el-table-column>
                <el-table-column label="Status" prop="status">
                	<template slot-scope="scope">
                		<div v-if="scope.row.status == '2'" class="active-status-div"><i class="el-icon el-icon-status-terminate"></i> <%=rb.getString("YiJieShu")%></div>
                		<div v-if="scope.row.status == '1'" class="active-status-div"><i class="el-icon el-icon-status-inProgress"></i> <%=rb.getString("JinXingZhong")%></div>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("JieGuo")%>" prop="result">
                	<template slot-scope="scope">
                		<div v-if="scope.row.result == '1'" class="active-status-div"><i class="el-icon el-icon-status-success success-status"></i> <%=rb.getString("ChengGong")%></div>
                		<div v-if="scope.row.result == '0'" class="active-status-div"><i class="el-icon el-icon-status-failed"></i> <%=rb.getString("ShiBai")%></div>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failureReason"></el-table-column>
                <el-table-column label="APN Policy" prop="apnPolicyName">
                    <template slot-scope="scope">
                        <el-popover>
                            <span slot="reference">
                                {{scope.row.apnPolicyName}}
                                <span v-if="scope.row.apnInfo.length>1">
                                    [ <span style="color:#4d84ff;">{{scope.row.apnInfo.length}}</span> 
                                    <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
                                </span>
                            </span>
                            <div class="apn-list">
                                <span class="apn-title">APN Policy Name: </span>{{scope.row.apnPolicyName}}
                                <div v-for="item in scope.row.apnInfo">
                                    <span class="apn-title">APN{{item.apnOrder}}</span> APN Name: {{item.apnName}}
                                </div>
                            </div>
                        </el-popover>
                    </template>
                </el-table-column>
                <el-table-column label="L2 Tunnel Enable" prop="l2TunnelEnable"></el-table-column>
                <el-table-column label="<%=rb.getString("CaoZuoShiJian")%>" prop="operationTime"></el-table-column>
            </el-ctable>
        </el-tab-pane>

        <el-tab-pane label="CPE">
            <el-ctable ref="cpe" id="device_cpe_policy" :time="6"
                :url="cpeURL" :query-params="cpeParams" 
                :query-params="cpeParams"
                @load-success="cpeLoadSuccess">
                <template slot="toolbar">
                    <div style="display: flex; justify-content: space-between;">
                        <el-query type="normal" placeholder="Serial Number / CPE Name" @query="queryCpe"></el-query>
                        
                        <div class='device_count'>
                            <p class="suc_count"><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{resultCpe.successCount}}</span></p>
                            <p class="fail_count"><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{resultCpe.failCount}}</span></p>
                        </div>
                    </div>
                </template>
                
                <el-table-column label="<%=rb.getString("CPEBianMa")%>" prop="serial_number"></el-table-column>
                <el-table-column label="<%=rb.getString("CPEName")%>" prop="cpe_name"></el-table-column>
                <el-table-column label="<%=rb.getString("MACDiZhi")%>" prop="macaddress"></el-table-column>
                <el-table-column label="IMSI" prop="imsi"></el-table-column>
                <el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
                	<template slot-scope="scope">
                		<div v-if="scope.row.status == '2'" class="active-status-div"><i class="el-icon el-icon-status-terminate"></i> <%=rb.getString("YiJieShu")%></div>
                		<div v-if="scope.row.status == '1'" class="active-status-div"><i class="el-icon el-icon-status-inProgress"></i> <%=rb.getString("JinXingZhong")%></div>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("JieGuo")%>" prop="result">
                	<template slot-scope="scope">
                		<div v-if="scope.row.result == '1'" class="active-status-div"><i class="el-icon el-icon-status-success success-status"></i> <%=rb.getString("ChengGong")%></div>
                		<div v-if="scope.row.result == '0'" class="active-status-div"><i class="el-icon el-icon-status-failed"></i> <%=rb.getString("ShiBai")%></div>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failure_reason"></el-table-column>
                <el-table-column label="APN Policy" prop="apn_policy_name">
                    <template slot-scope="scope">
                        <el-popover>
                            <span slot="reference">
                                {{scope.row.apn_policy_name}}
                                <span v-if="scope.row.apn_info.length>1">
                                    [ <span style="color:#4d84ff;">{{scope.row.apn_info.length}}</span> 
                                    <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
                                </span>
                            </span>
                            <div class="apn-list">
                                <span class="apn-title">APN Policy Name: </span>{{scope.row.apn_policy_name}}
                                <div v-for="item in scope.row.apn_info">
                                    <span class="apn-title">APN{{item.apnOrder}}</span> APN Name: {{item.apnName}}
                                </div>
                            </div>
                        </el-popover>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("CaoZuoShiJian")%>" prop="operation_time"></el-table-column>
            </el-ctable>
        </el-tab-pane>
    </el-tabs>

    <el-slide ref="policy" class="no-padding"
        method="get"
        :footer="false"
        :title="policyslide.title"
        :url="policyslide.url"
        @cancel="closeDevicePolicy">
    </el-slide>
</div>

<script>
    new Vue({
        el: '#result_device_ctn',
        data() {

            return {
                enbURL: '${ctx}/epc/apnpolicyresult/queryApnPolicyResultEnb.action',
                enbParams: {
                    searchText: '',
                    timeZone: timeZone
                },
                cpeURL: '${ctx}/epc/apnpolicyresult/queryApnPolicyResultCpe.action',
                cpeParams: {
                    searchText: '',
                    timeZone: timeZone
                },

                policyslide: {
                    title: '',
                    url: ''
                },

                resultEnb: {
                    success: 0,
                    failure: 0
                },
                resultCpe: {
                    successCount: 0,
                    failCount: 0
                }
            }
        },
        computed: {
        	writable() {
        		return writableMap.eNbHalobImsi == true;
        	}
        },
        methods: {
            queryEnb(text) {
                this.enbParams.searchText = text;
            },
            queryCpe(text) {
                this.cpeParams.searchText = text;
            },
            loadSuccess(data) {
                var vm = this;

                if(data) {
                    Object.assign(vm.resultEnb, data.properties);
                }
            },
            cpeLoadSuccess(data) {
                var vm = this;

                if(data) {
                    Object.assign(vm.resultCpe, data.properties);
                }
            },
            closeDevice() {
                eventBus.$emit('hide-device')
            },
            closeDevicePolicy() {
                this.$refs.policy.hide();
                this.$refs.enb.refresh();
                this.$refs.cpe.refresh();
            },
            addDevicePolicy() {
                var vm = this;

                vm.policyslide.title = 'Add Devices Policy';
                vm.policyslide.url = '${ctx}/epc/apnpolicyapply/goApnPolicyApply.action';
                vm.$refs.policy.showSlide();
            }
        },
        mounted() {

            eventBus.$off('close-device-policy').$on('close-device-policy', this.closeDevicePolicy);
        }
    });
</script>
