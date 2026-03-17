<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
	.no-footer .el-card__footer {
        display: none;
    }
    .apn-details {
        min-width: 150px;
    }
    .apn-details-item {
        margin-top: 10px;
    }
    .apn-details-item:first-of-type {
        margin-top: 1px;
    }
    .apn-policy .card-details {
        height: 155px;
    }
    .unenable::before {
        color: #c6c7ca;
    }
    .default-apn {
        border: 1px solid #F2B354;
        color: #F2B354;
        border-radius: 3px;
        margin-left: 5px;
        padding: 0 5px;
    }
</style>

<div id="apn_ctn" style="height: 100%;display: flex;">
    <el-tabs v-model="activeName" @tab-click="tabClick">
        <!-- APN Config -->
        <el-tab-pane label="<%=rb.getString("APNPeiZhi")%>" name="config">
            <el-ctable ref="configlist"
                :url="configURL" :query-params="configParams" >
                <template slot="toolbar">
                    <div style="display: flex; justify-content: space-between;">
                        <el-query type="normal" placeholder="<%=rb.getString("APNMingCheng")%>" @query="queryApn"></el-query>
                        <div>
                            <i v-if="writable" class="el-icon el-icon-circle-add" style="margin-right: 10px;" @click="addConfig"></i>
                            <i class="el-icon el-icon-circle-close" style="margin-right: 10px;" @click="closeAPN"></i>
                        </div>
                    </div>
                </template>
                
                <el-table-column width="40">
                    <template slot-scope="scope">
                        <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("APNMingCheng")%>" prop="APN_NAME"></el-table-column>
                <el-table-column label="APN AMBR_UL" prop="APN_AMBR_UL"></el-table-column>
                <el-table-column label="APN AMBR_DL" prop="APN_AMBR_DL"></el-table-column>
                <el-table-column label="PDN Type" prop="PDN_TYPE">
                    <template slot-scope="scope">
                        <div v-if="scope.row.PDN_TYPE == '1'">IPv4</div>
                        <div v-if="scope.row.PDN_TYPE == '2'">IPv6</div>
                    </template>
                </el-table-column>
                <el-table-column v-if="false" label="GW IP Address" prop="GW_IP_ADDRESS"></el-table-column>
                <el-table-column label="QCI" prop="QCI"></el-table-column>
                <el-table-column label="<%=rb.getString("FanWei")%>" prop="IPPOOL_INFO">
                    <template slot-scope="scope">
                        <el-popover v-if="scope.row.IPPOOL_INFO && scope.row.IPPOOL_INFO.length">
                            <span slot="reference">
                                {{ scope.row.IPPOOL_INFO[0]['START_SERVED_PARTY_IPV4_ADDRESS'] }} ~ {{ scope.row.IPPOOL_INFO[0]['END_SERVED_PARTY_IPV4_ADDRESS'] }}
                                <span v-if="scope.row.IPPOOL_INFO.length>1">[ <span style="color:#4d84ff;">{{scope.row.IPPOOL_INFO.length}}</span> ]</span>
                            </span>
                            <div class="apn-list">
                                <div v-for="item in scope.row.IPPOOL_INFO">
                                    {{item['START_SERVED_PARTY_IPV4_ADDRESS']}} ~ {{item['END_SERVED_PARTY_IPV4_ADDRESS']}}
                                </div>
                            </div>
                        </el-popover>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("CaoZuoShiJian")%>" prop="OPER_TIME"></el-table-column>
            </el-ctable>

            <el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>
        </el-tab-pane>
        <!-- APN Policy -->
        <el-tab-pane label="APN Policy" name="policy">
            <el-ctable ref="policylist" type="card" class="no-footer apn-policy"
                row-key="apnPolicyName"
                :card-option="cardOption"
                :url="policyURL" :query-params="policyParams"
                @card-menu-click="cardMenuClick">
                <template slot="toolbar">
                    <div style="display: flex; justify-content: space-between;">
                        <el-query type="normal" placeholder="APN Policy Name" @query="queryPolicy"></el-query>
                        <div>
                            <i v-if="writable" class="el-icon el-icon-circle-add" style="margin-right: 10px;" @click="addPolicy"></i>
                            <i class="el-icon el-icon-circle-close" style="margin-right: 10px;" @click="closeAPN"></i>
                        </div>
                    </div>
                </template>
            </el-ctable>
        </el-tab-pane>
    </el-tabs>

    <el-slide ref="slide" class="no-padding"
        method="get"
        :title="apnslide.title"
        :url="apnslide.url"
        :footer="apnslide.footer"
        @ok="saveAdd"
        @cancel="closeAdd">
    </el-slide>
</div>

<script>
new Vue({
    el: '#apn_ctn',
    data() {
        var vm = this,
            apnNameFmt = function(val, row) {
                var str = ''

                if(row.apnConfig) {
                    row.apnConfig.map(function(item){
                        if(item.defaultApn == 'true') {
                            str += '<p class="apn-details-item">APN ' + item.apnOrder + ': ' + item.apnName + '<span class="default-apn">Default</span></p>';
                        }else {
                            str += '<p class="apn-details-item">APN ' + item.apnOrder + ': ' + item.apnName + '</p>';
                        }
                    });
                }

                return '<div class="apn-details">'+str+'</div>';
            },
            enbTunnelFmt = function(val, row) {
                var str = '';

                if(val == 'true') {
                    str = '<span class="el-icon el-icon-status-enable"></span> True';
                }else if(val == 'false'){
                    str = '<span class="el-icon el-icon-status-enable unenable"></span> False';
                }

                return str;
            },
            modeFmt = function(val, row) {
                var html = '',
                    models = {
                        0: 'NAT',
                        1: 'Router Mode',
                        2: 'Tunnel Mode',
                        3: 'Bridge Mode'
                    };

                if(models[val]) {
                    html = models[val];
                }

                return html;
            },
            cardMenus = [{cls:"el-icon-operation-info el-icon",code:'info'}];
            
        if(writableMap.CODE_ENB_DEVICE_HALOB == true) {
        	cardMenus.push({cls:"el-icon-operation-edit el-icon" ,code:'edit'});
        	cardMenus.push({cls:"el-icon-operation-delete el-icon" ,code:'delete'});
        }

        return {
            activeName: '',

            configURL: '${ctx}/epc/apnconfig/getGwApnInfos.action',
            configParams: {
                searchText: '',
                timeZone: timeZone
            },

            policyURL: '${ctx}/epc/apnpolicyconfig/queryApnPolicyPageList.action',
            policyParams: {
                timeZone: timeZone,
                searchText: ''
            },

            menus: [],

            apnslide: {
                title: '',
                url: '',
                footer: true
            },
            cardOption: {
                fields: [
                    {label: 'apnPolicyName', field: 'apnPolicyName'},
                    {label: 'APN', field: 'apnPolicyName', formatter: apnNameFmt, labelShow: false},
                    {label: 'eNB L2 Tunnel Enable', field: 'enbL2TunnelEnable', formatter: enbTunnelFmt},
                    {label: 'CPE L2 Tunnel Mode', field: 'cpeL2TunnelMode', formatter: modeFmt}
                ],
                menus: cardMenus
            }
        };
    },
    computed: {
    	writable() {
    		return writableMap.CODE_ENB_DEVICE_HALOB == true;
    	}
    },
    methods: {
    	tabClick(tab) {

        },
        queryApn(txt) {
            this.configParams.searchText = txt;
        },
        queryPolicy(txt) {
            this.policyParams.searchText = txt;
        },
        optClick(row,ev) {
            var vm = this,
                modDisabled = false;

            if(row.OMC_DEFAULT == '1') {
                modDisabled = true;
            }

            vm.menus = [
                {label:'Information',cls:"el-icon-operation-info el-icon",code:'info', row: row}
            ];
            
            if(vm.writable) {
            	vm.menus.push({label:'Modify',cls:"el-icon-operation-edit el-icon" ,code:'edit', row: row, disable: modDisabled});
            }
            
            vm.$nextTick(function() {
                document.body.click();
                vm.$refs.menu.show(ev);
            });
        },
        handerClose() {
            this.$refs.menu.hide();
        },
        menuClick(row) {
            var vm = this,
                actions = {
                    info: vm.viewApn,
                    edit: vm.editApn
                },
                code = row.code;

            if(actions[code]) {
                actions[code](row.row);
            }
        },
        editApn(row) {
            var vm = this;

            vm.apnslide.title = "<%=rb.getString("XiuGaiAPN")%>";
            vm.apnslide.url = '${ctx}/epc/apnconfig/toApnConfigAdd.action';
            vm.apnslide.footer = true;
            vm.$refs.slide.showSlide(function(){
                eventBus.$emit('init-config', 'edit', row);
            });
        },
        viewApn(row) {
            var vm = this;

            vm.apnslide.title = "<%=rb.getString("ChaKanAPN")%>";
            vm.apnslide.url = '${ctx}/epc/apnconfig/toApnConfigAdd.action';
            vm.apnslide.footer = false;
            vm.$refs.slide.showSlide(function(){
                eventBus.$emit('init-config', 'view', row);
            });
        },
        cardMenuClick(menuRow, row) {
            var vm = this,
                code = menuRow.code;

            vm.addPolicy(code, row);
        },
        addConfig() {
            var vm = this;

            vm.apnslide.title = '<%=rb.getString("TianJiaAPN")%>';
            vm.apnslide.url = '${ctx}/epc/apnconfig/toApnConfigAdd.action';
            vm.apnslide.footer = true;
            vm.$refs.slide.showSlide();
        },
        closeAdd() {
            this.$refs.slide.hide();
        },
        closeAPN() {
            eventBus.$emit('hide-apn');
        },
        deletePolciy(row) {
            var vm = this,
                params = {
                    apnPolicyName: row.apnPolicyName
                };

            vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>', '<%=rb.getString("QueRen")%>').then(function(r){
                if(r) {
                    axios.post('${ctx}/epc/apnpolicyconfig/deleteApnPolicy.action',stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            vm.$message({
                                message: 'Success',
                                type: 'success'
                            });

                            vm.$refs.policylist.refresh();
                        }else {
                            vm.$message({
                                message: data.message,
                                type: 'error'
                            });
                        }
                    });
                }
            }).catch(function(){});
        },
        addPolicy(code, row) {
            var vm = this;

            vm.apnslide.title = '<%=rb.getString("TianJiaAPNPolicy")%>';
            vm.apnslide.footer = true;

            if(code == 'delete') {
                vm.deletePolciy(row);
                return;
            }
            
            if(code == 'info') {
                vm.apnslide.title = '<%=rb.getString("ChaKanAPNPolicy")%>';
                vm.apnslide.footer = false;
            }
            if(code == 'edit') {
                vm.apnslide.title = '<%=rb.getString("XiuGaiAPNPolicy")%>';
            }

            vm.apnslide.url = '${ctx}/epc/apnpolicyconfig/toApnPolicyAdd.action';
            vm.$refs.slide.showSlide(function(){
                eventBus.$emit('init-policy', code, row);
            });
        },
        saveAdd() {
            var vm = this;

            vm.activeName == 'config' && eventBus.$emit('add-config');
            vm.activeName == 'policy' && eventBus.$emit('add-policy');
        },
        refreshList() {
            var vm = this;

            vm.activeName == 'config' && vm.$refs.configlist.refresh();
            vm.activeName == 'policy' && vm.$refs.policylist.refresh();

            vm.closeAdd();
        }
    },
    mounted() {
    	var vm = this;
    	
    	eventBus.$off('refresh-list').$on('refresh-list', this.refreshList);
    	setTimeout(function(){
    		vm.activeName = 'config';
    	},200);
    }
});
</script>
