<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
    .margin-left-title {
        margin: 5px 0 10px 25px;
    }
    .padding-left-px {
        padding-left: 50px;
    }
    .padding-left-px .el-form-item {
        display: inline-block;
        margin-right: 150px;
    }

    #config_add_ctn .el-form-item__label {
        line-height: 30px;
    }
    .splite-line {
        opacity: 0.2;
        margin-bottom: 30px;
    }
    .pool-item {
        position: relative;
        display: inline-block;
        padding: 5px;
        margin-bottom: 5px;
        margin-right: 5px;
        max-width: 220px;
        border: 1px solid #4d84ff;
    }
    .pool-start, .pool-end {
        display: inline-block;
        min-width: 90px;
    }
    .pool-end {
        margin-right: 25px;
    }
    .error {
        color: red;
    }
</style>

<div id="config_add_ctn" style="padding-top: 30px;">
    <el-form ref="form" :model="form" :rules="rules" label-position="left" label-width="150">
        <div class="group-title not-extend margin-left-title">
            <span class="title-icon"></span>
            <span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
        </div>
        <div class="padding-left-px">
            <el-form-item label="<%=rb.getString("APNMingCheng")%>" prop="APN_NAME">
                <el-input v-model="form.APN_NAME" :disabled="nameDisabled"></el-input>
            </el-form-item>
            <el-form-item label="APN AMBR_UL" prop="APN_AMBR_UL">
                <el-input v-model="form.APN_AMBR_UL" :disabled="readonly" type="number">
                    <span slot="append">Mbps</span>
                </el-input>
            </el-form-item>
            <el-form-item label="APN AMBR_DL" prop="APN_AMBR_DL">
                <el-input v-model="form.APN_AMBR_DL" :disabled="readonly" type="number">
                    <span slot="append">Mbps</span>
                </el-input>
            </el-form-item>
        </div>
        
        <hr class="splite-line">

        <div class="group-title not-extend margin-left-title">
            <span class="title-icon"></span>
            <span class="title-text">IP Alloc</span>
        </div>
        <el-form-item label="PDN Type" class="padding-left-px">
            <el-select v-model="form.PDN_TYPE" :disabled="readonly" @change="ipTypeChange">
                <el-option label="IPv4" value="1"></el-option>
                <el-option label="IPv6" value="2"></el-option>
            </el-select>
        </el-form-item>

        <div class="padding-left-px">
            <el-form-item :label="form.PDN_TYPE=='1'?'IPv4 Start':'IPv6 Start'">
                <el-input v-model="iprange.start" :disabled="form.ALLOCATION_TYPE=='dhcp' || readonly"></el-input>
            </el-form-item>
            <el-form-item :label="form.PDN_TYPE=='1'?'IPv4 End':'IPv6 End'" style="margin-right: 20px;">
                <el-input v-model="iprange.end" :disabled="form.ALLOCATION_TYPE=='dhcp' || readonly"></el-input>
            </el-form-item>
            <i class="el-icon el-icon-plus" v-show="!readonly"  v-if="form.ALLOCATION_TYPE == 'dhcp' || form.IPPOOL_INFO.length==3" style="opacity: 0.5;"></i> 
            <i class="el-icon el-icon-plus" v-show="!readonly" v-if="form.ALLOCATION_TYPE != 'dhcp' && form.IPPOOL_INFO.length<3" @click="addPool"></i> 
            <span v-show="!readonly" style="color: #4d84ff; display: inline-block; margin-left: 5px;">(No more than 3)</span>

            <div style="padding-left: 150px;margin-top: -20px;margin-bottom: 20px;">
                <div v-for="(item,index) in form.IPPOOL_INFO" class="pool-item">
                    <span class="pool-start">{{item.START_SERVED_PARTY_IPV4_ADDRESS}}</span>~
                    <span class="pool-end">{{item.END_SERVED_PARTY_IPV4_ADDRESS}}</span>
                    <span v-show="isHistoryIP(item)" class="el-icon el-icon-close" style="top: 6px; right: 5px; font-size: 15px;position: absolute;" @click="removePool(index)"></span>
                </div>
            </div>
            <div v-show="poolMsg != ''" class="error" style="padding-left: 150px;margin-top: -25px;margin-bottom: 20px;">
                {{poolMsg}}
            </div>
            <el-form-item v-show="poolMsg == ''" prop="IPPOOL_INFO" style="margin-top: -25px;">
                <el-input v-show="false" v-model="form.IPPOOL_INFO"></el-input>
            </el-form-item>
        </div>

        <hr class="splite-line">

        <div class="group-title not-extend margin-left-title">
            <span class="title-icon"></span>
            <span class="title-text">DNS</span>
        </div>
        <div class="padding-left-px">
            <el-form-item label="Primary DNS IP" prop="PRIMARY_DNS_IPADDR">
                <el-input v-model="form.PRIMARY_DNS_IPADDR" :disabled="readonly"></el-input>
            </el-form-item>
            <el-form-item label="Secondary DNS IP" prop="SECONDARY_DNS_IPADDR">
                <el-input v-model="form.SECONDARY_DNS_IPADDR" :disabled="readonly"></el-input>
            </el-form-item>
        </div>

        <hr class="splite-line">

        <div class="group-title not-extend margin-left-title">
            <span class="title-icon"></span>
            <span class="title-text">GW</span>
        </div>
        <div class="padding-left-px">
            <el-form-item label="<%=rb.getString("LgwIPAddress")%>" prop="GW_IP_ADDRESS">
                <el-input v-model="form.GW_IP_ADDRESS" :disabled="readonly"></el-input>
            </el-form-item>
        </div>

        <hr class="splite-line">

        <div class="group-title not-extend margin-left-title">
            <span class="title-icon"></span>
            <span class="title-text">QoS</span>
        </div>
        <div class="padding-left-px">
            <el-form-item label="QCI" prop="QCI">
                <el-input v-model="form.QCI" type="number" :disabled="readonly"></el-input>
            </el-form-item>
            <el-form-item label="ARP Priority Level" prop="ARP_PRIORITYLEVEL">
                <el-input v-model="form.ARP_PRIORITYLEVEL" type="number" :disabled="readonly"></el-input>
            </el-form-item>
            <el-form-item label="ARP PCI">
                <el-select v-model="form.ARP_PCI" :disabled="readonly">
                    <el-option label="Close" value="0"></el-option>
                    <el-option label="Open" value="1"></el-option>
                </el-select>
            </el-form-item>
        </div>
        <el-form-item label="ARP PVI" class="padding-left-px">
            <el-select v-model="form.ARP_PVI" :disabled="readonly">
                <el-option label="Close" value="0"></el-option>
                <el-option label="Open" value="1"></el-option>
            </el-select>
        </el-form-item>

    </el-form>
</div>

<script>
    new Vue({
        el: '#config_add_ctn',
        data() {
            var vm = this,
                validIP = function(rule, val, cb) {
                    if(val && isIPv4(val)) {
                        cb();
                    }else if(val) {
                        cb('<%=rb.getString("IPDiZhi")%>');
                    }else {
                        cb('<%=rb.getString("BiTian")%>');
                    }
                },
                validNum = function(rule, val, cb) {
                    if(val-rule.min<0 || val-rule.max>0) {
                        cb('Number: '+ rule.min + '-' + rule.max);
                    }else {
                        cb();
                    }
                };

            return {
                type: 'add',
                oldPOOLINFO: [],
                form: {
                    APN_NAME: '',
                    QCI: '',
                    ALLOCATION_TYPE: 'dynamic',
                    GW_IP_ADDRESS: '',
                    APN_AMBR_UL: '',
                    APN_AMBR_DL: '',
                    PRIMARY_DNS_IPADDR: '',
                    SECONDARY_DNS_IPADDR: '',
                    ARP_PRIORITYLEVEL: '',
                    ARP_PCI: '0',
                    ARP_PVI: '0',
                    PDN_TYPE: '1',
                    IPPOOL_INFO: [],
                    eDRX_PTW: '',
                    eDRX_VALUE: '',
                    T_TIMER: '',
                    eT_TIMER: '',
                    As_IP: ''
                },
                rules: {
                    APN_NAME: [{required: true, message: 'Required'}],
                    APN_AMBR_UL: [{required: true, message: 'Required'}],
                    APN_AMBR_DL: [{required: true, message: 'Required'}],
                    QCI: [{validator: validNum, message: 'Number 5-9',max:9,min:5}],
                    ARP_PRIORITYLEVEL: [{validator: validNum, message: 'Number 1-14',max:14,min:1}],
                    GW_IP_ADDRESS: [{validator: validIP}],
                    PRIMARY_DNS_IPADDR: [{validator: validIP}],
                    SECONDARY_DNS_IPADDR: [{validator: validIP}],
                    IPPOOL_INFO: [{required: true, message: 'Required'}]
                },
                poolMsg: '',
                iprange: {
                    start: '',
                    end: ''
                },
                readonly: false
            };
        },
        computed: {
            nameDisabled() {
                return ['view', 'edit'].includes(this.type);
            }
        },
        methods: {
            init(type, row) {
                var vm = this;

                if(['view', 'edit'].includes(type)) {
                    vm.type = type;

                    var params = {
                            apnName: row.APN_NAME
                        };

                    if('view' == type) {
                        vm.readonly = true;
                    }else {
                        vm.readonly = false;
                    }

                    axios.post('${ctx}/epc/apnconfig/queryApnInfoByName.action', stringify(params)).then(function(res){
                        var data = res.data || {};

                        Object.assign(vm.form, data);

                        vm.form.IPPOOL_INFO.map(function(item){
                            vm.oldPOOLINFO.push(Object.assign({},item));
                        });
                    });
                }
            },
            ipTypeChange(val) {
                this.form.IPPOOL_INFO = [];
            },
            leftPadZero(text, max) {
            	var length = text.length,
            		dis = max - length;
            	
            	if(dis) {
            		for(var i = 0; i<dis; i++) {
            			text = '0' + text;
            		}
            	}
            	
            	return text;
            },
            isHistoryIP(item) {
                var vm = this,
                    show = true;

                if(['view'].includes(this.type)) {
                    show = false;
                }else {
                    vm.oldPOOLINFO.map(function(m){
                        var ip = m.START_SERVED_PARTY_IPV4_ADDRESS +'-'+ m.END_SERVED_PARTY_IPV4_ADDRESS,
                            itemIp = item.START_SERVED_PARTY_IPV4_ADDRESS +'-'+ item.END_SERVED_PARTY_IPV4_ADDRESS;

                        if(ip == itemIp) {
                            show = false;
                        }
                    });
                }

                return show;
            },
            validRange(start, end) {
                var vm = this,
                	spit = vm.form.PDN_TYPE == 1? '.':':',
                	max = vm.form.PDN_TYPE == 1?3:4,
                    preList = start.split(spit),
                    sufList = end.split(spit),
                    isLess = true;

                var startIPStr = preList.map(function(item, idx) {
	                    return vm.leftPadZero(item, max);
	                }).join(''),
	                endIPStr = sufList.map(function(item, idx) {
	                    return vm.leftPadZero(item, max);
	                }).join('');
				
                if(startIPStr > endIPStr || vm.isCovered(startIPStr, endIPStr)) isLess = false;
                
                return isLess;
            },
            isCovered(start, end) {
                var vm = this,
                    bool = false;

                vm.form.IPPOOL_INFO.map(function(item){
                    var sIp = item.START_SERVED_PARTY_IPV4_ADDRESS,
                        eIp = item.END_SERVED_PARTY_IPV4_ADDRESS,
                        spit = vm.form.PDN_TYPE == 1? '.':':',
                        max = vm.form.PDN_TYPE == 1?3:4,
                        preList = sIp.split(spit),
                        sufList = eIp.split(spit);
                    
                    var startIPStr = preList.map(function(item) {
	                    return vm.leftPadZero(item, max);
	                }).join(''),
	                endIPStr = sufList.map(function(item) {
	                    return vm.leftPadZero(item, max);
	                }).join('');

                    if((start<startIPStr && end<endIPStr) || (start>startIPStr && end>endIPStr)) {

                    }else {
                        bool = true;
                    }
                });

                return bool;
            },
            addPool() {
                var vm = this,
                    sip = vm.iprange.start,
                    eip = vm.iprange.end,
                    list = vm.form.IPPOOL_INFO.map(function(item){
                        return item.START_SERVED_PARTY_IPV4_ADDRESS +'-'+ item.END_SERVED_PARTY_IPV4_ADDRESS;
                    }),
                    validators = {
                        1: isIPv4,
                        2: isIPv6
                    },
                    validFn = validators[vm.form.PDN_TYPE];

                if(validFn(sip) && validFn(eip) && !list.includes(sip+'-'+eip) && vm.validRange(sip,eip)) {
                    vm.form.IPPOOL_INFO.push({
                        START_SERVED_PARTY_IPV4_ADDRESS: vm.iprange.start,
                        END_SERVED_PARTY_IPV4_ADDRESS: vm.iprange.end
                    });
                    
                    Object.assign(vm.iprange, {
                    	start: '',
                    	end: ''
                    });
                }else {
                    vm.poolMsg = '<%=rb.getString("IPFangWeiTiShi")%>';
                    
                    setTimeout(function(){
                        vm.poolMsg = '';
                    },3000);
                }
            },
            removePool(idx) {
                this.form.IPPOOL_INFO.splice(idx,1);
            },
            submit() {
                var vm = this,
                    url = '${ctx}/epc/apnconfig/addGwApnInfos.action',
                    params = {

                    };

                if(vm.type == 'edit') {
                    url = '${ctx}/epc/apnconfig/updateGwApnInfos.action';
                }

                Object.assign(params, vm.form);
                params.IPPOOL_INFO = JSON.stringify(params.IPPOOL_INFO);

                vm.$refs.form.validate(function(r){
                    if(r) {
                        axios.post(url,stringify(params)).then(function(res){
                            var data = res.data;
                            
                            if(data.success == true) {
                                vm.$message({
                                    message: '<%=rb.getString("ChengGong")%>',
                                    type: 'success'
                                });
                                
                                eventBus.$emit('refresh-list', 'config');
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
            eventBus.$off('add-config').$on('add-config', this.submit);
            eventBus.$off('init-config').$on('init-config', this.init);
        }
    });

</script>
