<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
    .padding-left-px {
        padding-left: 60px;
    }
    .padding-left-px .el-form-item {
        display: inline-block;
        margin-right: 150px;
    }

    #policy_add_ctn .el-form-item__label {
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

    .select-80px .el-input__inner,
    .select-height-26  .el-input__inner{
        max-height: 26px;
        height: 26px;
    }
    .select-80px .el-input {
        width: 80px;
    }
</style>

<div id="policy_add_ctn" style="padding-top: 30px;">
    <el-form ref="form" :model="form" :rules="rules" label-position="left" label-width="150">
        <div class="group-title not-extend margin-left-px">
            <span class="title-icon"></span>
            <span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
        </div>
        <el-form-item label="APN Policy Name" class="padding-left-px" prop="apnPolicyName">
            <el-input v-model="form.apnPolicyName" :disabled="nameDisabled"></el-input>
        </el-form-item>

        <hr class="splite-line">

        <div class="group-title not-extend margin-left-px">
            <span class="title-icon"></span>
            <span class="title-text">APN List</span>
        </div>
        <div class="padding-left-px">
            <div v-for="(row,idx) in form.apnConfig">
                <el-checkbox v-model="form.apnConfig[idx].apnEnable" :disabled="readonly" style="margin-right: 20px;" 
                    @change="function(val){
                        if(val == 'true') {
                            form.apnConfig.map(function(m){
                                if(m.apnOrder != row.apnOrder && m.defaultApn == '1') {
                                    row.defaultApn = '0';
                                }
                            });
                        }

                        enableChange(val);
                    }"
                    true-label="true" false-label="false">
                </el-checkbox>

                <span class="prefix-title">APN{{idx+1}}</span>
                <el-form-item label="APN Name" label-width="100" style="margin-right: 20px;"
                    :prop="'apnConfig.'+idx+'.apnName'"
                    :key="row.apnOrder"
                    :rules="{
                        validator: function(rule, val, cb){
                            var isRepeated = false,
                                enableList = form.apnConfig.filter(function(item){
                                    return item.apnEnable == 'true';
                                }),
                                nameList = [];
                            
                            enableList.filter(function(item){
                                if(item.apnName == val) {
                                    nameList.push(item.apnName);
                                }
                            });

                            if(nameList.length > 1) {
                                isRepeated = true;
                            }

                            if(isRepeated && val) {
                                cb('Repeated')
                            }else {
                                cb();
                            }
                        }
                    }"
                    >
                    <el-select v-model="form.apnConfig[idx].apnName" class="select-height-26" 
                        :disabled="form.apnConfig[idx].apnEnable!='true' || readonly">
                        <el-option v-for="item in apnList" :label="item.label" :value="item.value"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item label="eNB Default APN" label-width="140" style="margin-right: 50px;"
                    :prop="'apnConfig.'+idx+'.defaultApn'"
                    :key="row.apnOrder+'_def'"
                    :rules="{
                        validator: function(rule, val, cb){
                            if(row.apnEnable == 'true' && row.defaultApn == '1') {
                                form.apnConfig.map(function(m){
                                    if(m.apnEnable == 'true' && m.apnOrder != row.apnOrder) m.defaultApn = '0';
                                });
                            }
                            
                            var defaultList = form.apnConfig.filter(function(m){
                                    return m.apnEnable == 'true' && m.defaultApn == '1';
                                }),
                                hasDefault = defaultList.length>0;
                            
                            if(hasDefault || readonly) {
                                cb();
                            }else {
                                if(row.apnEnable == 'false') {
                                    cb();
                                }else {
                                    cb('No default');
                                }
                            }
                        }
                    }">
                    <el-select v-model="form.apnConfig[idx].defaultApn" class="select-80px apn-default" 
                        @change="function(val){
                            form.apnConfig.map(function(item){
                                if(item.bearType == '1') item.bearType = '0';
                            });
                            if(val == '1') {
                                row.bearType = '1';
                                row.greType = '0';
                            }

                            defaultChange(val);
                        }"
                        :disabled="form.apnConfig[idx].apnEnable!='true' || readonly">
                        <el-option label="True" value="1"></el-option>
                        <el-option label="False" value="0"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item label="Bearer Type" label-width="100" style="margin-right: 50px;">
                    <el-select v-model="form.apnConfig[idx].bearType" class="select-80px" 
                        @change="function(val){
                            if(val == '1') {
                                row.defaultApn = '1';
                            }

                            if(val == '0') {
                                row.greType = '1';
                            }else {
                                row.vlanId = '';
                                row.greType = '0';
                            }

                            validConfig();
                        }"
                        :disabled="form.apnConfig[idx].apnEnable!='true' || readonly">
                        <el-option label="MGMT" value="1" :disabled="mgmtDisabled"></el-option>
                        <el-option label="VOIP" value="2"></el-option>
                        <el-option label="DATA" value="0"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item label="APN Type" label-width="100" style="margin-right: 50px;">
                    <el-select v-model="row.greType" class="select-80px" :disabled="row.defaultApn=='1'"
                        @change="function(val){
                            validConfig();
                        }">
                        <el-option label="Layer2" value="1"></el-option>
                        <el-option label="Layer3" value="0"></el-option>
                    </el-select>
                </el-form-item>
                
                <el-form-item :label="row.greType == '0'?'VLAN ID':'VLAN List'" label-width="80" style="margin-right: 50px; width: 325px;"
                    :prop="'apnConfig.'+idx+'.vlanId'"
                    :key="row.apnOrder+'_vlanId'"
                    :rules="{
                        validator: function(rule, val, cb){
                            var reg = /^\d{1,}$/,
                                ids = (val||'').split(','),
                                bool = true;

                            ids.map(function(item){
                                var trimVal = item.trim();
                            	
                                if(!reg.test(trimVal)) {
                                    bool = false;
                                }
                                
                                if(trimVal<0 || trimVal>4094 || trimVal == '1') {
                                	bool = false;
                                }
                            });
                            
                            if(row.apnEnable=='true' && !readonly) {
                                if(row.greType == '1' && (row.apnEnable=='true' && !bool || ids.length>3)) {
                                	if(val) cb('Numbers separated by commas,max 3,range: 0-4094 except 1');
                                	else cb();
                                }else if(row.greType == '0' && (row.apnEnable=='true' && !bool || ids.length>1)) {
                                    if(val) {
                                        cb('Numbers,range: 0-4094 except 1');
                                    }else {
                                        var isL3Empty = false,
                                            l3LanId = form.apnConfig.filter(function(m){
                                                return m.greType == '0'
                                            }),
                                            l3EmptyLanId = form.apnConfig.filter(function(m){
                                                return m.greType == '0' && m.vlanId.trim() == '';
                                            });

                                        if(row.greType == '0' && l3EmptyLanId.length == l3LanId.length) {
                                            cb('VLAN ID of Layer3 should not all empty');
                                        }else {
                                            cb();
                                        }
                                    }
                                }else {
                                    var enableVlans = form.apnConfig.filter(function(m){
                                            return m.apnEnable == 'true' && m.vlanId != '' && m.apnOrder != (idx+1);
                                        }),
                                        otherVlans = [],
                                        selfVlans = [],
                                        isRepeated = false;

                                    enableVlans.map(function(o){
                                        o.vlanId.split(',').map(function(m){
                                            otherVlans.push(m);
                                        });
                                    });

                                    ids.map(function(m){
                                        if(!selfVlans.includes(m)) {
                                            selfVlans.push(m);
                                        }
                                    });

                                    ids.map(function(m){
                                        if(otherVlans.includes(m)) isRepeated = true;
                                    });

                                    if(ids.length > selfVlans.length) isRepeated = true;

                                    if(isRepeated && row.apnEnable == 'true') {
                                        cb('Vlanid repeated');
                                    }else {
                                        cb();
                                    }
                                }  
                            }else {
                                cb();
                            }
                        }
                    }">
                    <el-input v-model="form.apnConfig[idx].vlanId" :disabled="form.apnConfig[idx].apnEnable!='true' || readonly"></el-input>
                </el-form-item>
            </div>
            <el-form-item prop="apnConfig" label-width="0" style="width: 400px;">
                <el-input v-show="false" v-model="form.apnConfig"></el-input>
            </el-form-item>
        </div>

        <hr class="splite-line">

        <div class="group-title not-extend margin-left-px">
            <span class="title-icon"></span>
            <span class="title-text">Halob L2 Tunnel</span>
        </div>
        <div class="padding-left-px">
            <el-form-item label="L2 Tunnel Enable">
                <el-switch v-model="form.enbL2TunnelEnable" :disabled="readonly" active-value="true" inactive-value="false"></el-switch>
            </el-form-item>
            <el-form-item label="L2 Server IP" prop="enbL2ServerIp">
                <el-input v-model="form.enbL2ServerIp" :disabled="readonly"></el-input>
            </el-form-item>
            <el-form-item label="L2 Tunnel Mode">
                <el-select v-model="form.enbL2TunnelMode" disabled>
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
                <el-select v-model="form.cpeL2TunnelMode" disabled>
                    <el-option label="NAT" value="0"></el-option>
                    <el-option label="Router Mode" value="1"></el-option>
                    <el-option label="Tunnel Mode" value="2"></el-option>
                    <el-option label="Bridge Mode" value="3"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label="GRE Type">
                <el-select v-model="form.greType" disabled>
                    <el-option label="Layer2" value="1"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label="Destination IP">
                <el-input v-model="form.destinationIp" disabled></el-input>
            </el-form-item>
        </div>
		<div class="padding-left-px" style="font-weight: bold; padding-bottom: 20px;">
            WAN Config
        </div>
        <template v-for="(row,idx) in form.apnConfig" v-if="row.apnEnable == 'true'">
            <div class="padding-left-px">
                <span style="font-weight: bold;display: inline-block;margin-right: 15px;">APN{{row.apnOrder}}</span>
                <el-form-item label="Bearer Type">
                    <el-select v-model="row.bearType" disabled>
                        <el-option label="MGMT" value="1"></el-option>
                        <el-option label="VOIP" value="2"></el-option>
                        <el-option label="DATA" value="0"></el-option>
                    </el-select>
                </el-form-item>
            </div>
        </template>
    </el-form>
</div>

<script>
    new Vue({
        el: '#policy_add_ctn',
        data() {
            var vm = this,
                validPolicyName = function(rule,value,cb) {
                    var reg = /^[a-zA-Z0-9]{1,}$/g,
                        params = {
                            policyName: value   
                        };
                    
                    if(vm.nameDisabled) {
                        cb();
                    }else {
                        if(reg.test(value)) {
                            axios.post('${ctx}/epc/apnpolicyconfig/checkPolicyNameExist.action',stringify(params)).then(function(res) {
                                var result = res.data;

                                if(result.success) {
                                    cb('<%=rb.getString("MingChengYiCunZai")%>');
                                }else {
                                    cb();
                                }
                            });
                        }else {
                            cb('Required, character or number');
                        }
                    }
                },
                validServerIP = function(rule,value,cb) {
                    if(value && !isIPv4(value)) {
                        cb('<%=rb.getString("IPDiZhi")%>');
                    }else if(value){
                        cb();
                    }else {
                        if(vm.form.enbL2TunnelEnable == 'true') {
                            cb('Required');
                        }else {
                            cb();
                        }
                    }
                },
                validConfig = function(rule,value,cb) {
                    var enableItems = (value||[]).filter(function(item){
                            return item.apnEnable == 'true';
                        }),
                        isOver2 = false,
                        isBearTypeRight = false,
                        bearTypes = enableItems.map(function(item){
                            return item.bearType;
                        });
                    
                    if(enableItems.length>=2) isOver2 = true;
                    if(bearTypes.includes('1')) isBearTypeRight = true;

                    if(isBearTypeRight) {
                        cb();
                    }else {
                        cb('<%=rb.getString("XuanZeBearType")%>');
                    }
                };

            return {
                type: 'add',
                readonly: false,
                form: {
                    apnPolicyName: '',
                    apnName: '',
                    enbL2TunnelEnable: 'false',
                    enbL2ServerIp: '',
                    enbL2TunnelMode: '2',
                    cpeL2TunnelMode: '2',
                    destinationIp: '',
                    greType: '1',
                    apnConfig: [
                        {
                            apnOrder: 1,
                            apnEnable: 'true',
                            defaultApn: '1',
                            bearType: '1',
                            greType: '0',
                            apnName: 'mgmt.bc',
                            dhcpEnable: '1',
                            vpnType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        },
                        {
                            apnOrder: 2,
                            apnEnable: 'true',
                            defaultApn: '0',
                            bearType: '2',
                            greType: '0',
                            apnName: 'voip.bc',
                            dhcpEnable: '1',
                            vpnType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        },
                        {
                            apnOrder: 3,
                            apnEnable: 'true',
                            defaultApn: '0',
                            bearType: '0',
                            greType: '1',
                            apnName: 'data-high.bc',
                            dhcpEnable: '1',
                            vpnType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        },
                        {
                            apnOrder: 4,
                            apnEnable: 'true',
                            defaultApn: '0',
                            bearType: '0',
                            greType: '1',
                            apnName: 'data-low.bc',
                            dhcpEnable: '1',
                            vpnType: '1',
                            tunnelIpAddress: '',
                            tunnelSubnetMask: '',
                            vlanId: ''
                        }
                    ]
                },
                rules: {
                    apnConfig: [
                        {validator: validConfig}
                    ],
                    apnPolicyName: [
                        {validator: validPolicyName}
                    ],
                    enbL2ServerIp: [
                        {validator: validServerIP}
                    ]
                },
                apnList: []
            };
        },
        computed: {
            nameDisabled() {
                return ['info','edit'].includes(this.type);
            },
            mgmtDisabled() {
                var vm = this,
                    bool = false;

                vm.form.apnConfig.map(function(item){
                    if(item.bearType == '1' && item.apnEnable == 'true') bool = true;
                });

                return bool;
            }
        },
        watch: {
            'form.enbL2ServerIp': function(val) {
                this.form.destinationIp = val;
            }
        },
        methods: {
            init(code, row) {
                var vm = this;
                        
                vm.type = code;

                if(code == 'info') {
                    vm.readonly = true;
                }

                if(['info','edit'].includes(code)) {
                    var params = {
                            apnPolicyName: row.apnPolicyName
                        };

                    axios.post('${ctx}/epc/apnpolicyconfig/queryApnPolicyInfo.action', stringify(params)).then(function(res){
                        var data = res.data||{};

                        Object.assign(vm.form, data);
						
                        vm.$nextTick(function(){
                            vm.$refs.form.validate().then(()=>{}).catch(()=>{});
                        })
                    });
                }
            },
            getApnList() {
                var vm =this;

                axios.post('${ctx}/epc/apnconfig/queryApnNameList.action').then(function(res){
                    var names = res.data||[];
                    vm.apnList = names.map(function(item){
                        return {
                            label: item,
                            value: item
                        }
                    });
                });
            },
            enableChange(val) {
                var vm = this;
                
                vm.$refs.form.validate(function(){});
            },
            defaultChange(val) {
                var vm = this;

                vm.$nextTick(function(){
                    vm.$refs.form.validate('apnConfig').then(()=>{}).catch(()=>{});
                })
            },
            validConfig() {
                this.$refs.form.validate('apnConfig').then(()=>{}).catch(()=>{});
            },
            submit() {
                var vm = this,
                    url = '${ctx}/epc/apnpolicyconfig/addApnPolicy.action',
                    params = {

                    };

                if(this.type == 'edit') {
                    url = '${ctx}/epc/apnpolicyconfig/updateApnPolicy.action';
                }

                Object.assign(params, vm.form);
                
                params.apnConfig = JSON.stringify(params.apnConfig);

                vm.$refs.form.validate(function(r) {
                    if(r) {
                        axios.post(url, stringify(params)).then(function(res){
                            var data = res.data,
                            	tipMsg = vm.type == 'edit'?'<%=rb.getString("ChengGongDouHaoChongQiShengXiao")%>':'<%=rb.getString("ChengGong")%>';

                            if(data.success == true) {
                            	eventBus.$message({
                                    message: tipMsg,
                                    type: 'success'
                                });

                                eventBus.$emit('refresh-list', 'policy');
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
            this.getApnList();
            eventBus.$off('add-policy').$on('add-policy', this.submit);
            eventBus.$off('init-policy').$on('init-policy', this.init);
        }
    });

</script>
