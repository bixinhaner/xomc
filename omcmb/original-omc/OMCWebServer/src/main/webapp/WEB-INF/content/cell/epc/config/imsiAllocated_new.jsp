<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
    #imsi_allocated_ctn {
        height: 100%;
        width: 100%;
        position: relative;
    }
    #imsi_allocated_ctn .imsiAllocatedItemCls {
        height: 49%;
        width: 100%;
        position: relative;
    }
    #imsi_allocated_ctn .el-ctable {
        border-radius: 10px;
        box-shadow: 3px 3px 8px #d2d2d2;
    }
    #imsi_allocated_ctn .el-ctable .el-card__footer {
        display: none;
    }
    #imsi_allocated_ctn .el-ctable .card-details {
        display: flex;
        flex-direction: column;
        height: 240px;
    }
    #imsi_allocated_ctn .el-ctable .el-icon-tickets::before {
        content: 'APN';
        position: absolute;
        top: -3px;
        left: -30px;
        font-size: 14px;
        z-index: 100;
    }
     .el-tooltip__popper.is-dark { margin: 0 30px 0 80px; }
     .el-icon-status-SIM-avaliable1:before{
     	color: #4D84FF;
     }
     .el-icon-status-redirect:before{
     	color: #E88282;
     }
     .success-status::before {
        color: #67D972;
     }
     .no-border-tb .el-table--border td {
        border-right: none;
     }
     #imsi_allocated_ctn .imsiAllocatedMainBoxCls{
        width: 100%;
        height: 100%;
     }
     #imsi_allocated_ctn .apnAndMisisdBoxCls{
        width: 100%;
        height: 100%;
        display: flex;
     }
     #imsi_allocated_ctn .apnAndMisisdBoxCls > div:nth-child(1){
        height: 100%;
        flex:3;
     }
     #imsi_allocated_ctn .apnAndMisisdBoxCls > div:nth-child(2){
        height: 100%;
        margin-left: 10px;
        flex:2;
     }
     #imsi_allocated_ctn .el-col-5 {
        width: 30%;
        min-width:320px !important;
     }
     #imsi_allocated_ctn .toolbarHeadBoxCls{
        position: relative;
        display: flex; 
        justify-content: space-between;
     }
     #imsi_allocated_ctn .toolbarHeadBoxCls .el-input.el-input--small{
        width: 320px;
     }
     #imsi_allocated_ctn .toolbarHeadBtnCls{
        position: absolute;
        right: 20px;
        top: 5px;
        display: flex;
    }
    #imsi_allocated_ctn .toolbarHeadBtnCls > div{
        position: relative;
        margin-left: 10px;
    }
</style>

<div id="imsi_allocated_ctn" style="background: #F6F7FB;">
    <div class="imsiAllocatedMainBoxCls">
        <div class="imsiAllocatedItemCls" style="margin-bottom: 10px;">
            <!-- IMSI Allocated -->
            <el-ctable ref="imsi" id="imsi_allocted_list"
                :url="imsiURL"
                :query-params="imsiParams">
                <template slot="toolbar">
                    <div class="toolbarHeadBoxCls">
                        <div style="display: flex;align-items: center;">
                            <h3 style="padding-left: 10px;">IMSI Allocated</h3>
                            <el-query type="normal" placeholder="IMSI" @query="queryIMSI"></el-query>
                            <div class="toolbarHeadBtnCls">
                                <div class="newIconBoxCls-bt" @click="exportIMSI" tip="<%=rb.getString("DaoChu")%>">
                                    <span class="el-icon el-icon-circle-export"  ></span>
                                </div>
                                <div class="newIconBoxCls-bt" @click="closeSlide" tip="<%=rb.getString("GuanBi")%>">
                                    <span class="el-icon el-icon-circle-close"></span>
                                </div>
                            </div>
                        </div>
                    </div>
                </template>
                <el-table-column label="IMSI" prop="imsi"></el-table-column>
                <el-table-column label="<%=rb.getString("JiHuoZhuangTai")%>" prop="activeStatus">
                    <template slot-scope="scope">
                        <div v-if="scope.row.activeStatus == '0'" class="active-status-div"><i class="el-icon el-icon-status-de-active"></i> <%=rb.getString("WeiJiHuo")%></div>
                        <div v-if="scope.row.activeStatus == '1'" class="active-status-div"><i class="el-icon el-icon-status-active1 success-status"></i> <%=rb.getString("JiHuo")%></div>
                        <div v-if="scope.row.activeStatus == '2'" class="active-status-div"><i class="el-icon el-icon-status-redirect"></i> Redirect</div>
                        <div v-if="scope.row.activeStatus == '4'" class="active-status-div"><i class="el-icon el-icon-status-not-enabled"></i> <%=rb.getString("ZhuXiao")%></div>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="assignStatus">
                    <template slot-scope="scope">
                        <div v-if="scope.row.assignStatus == '0'" class="active-status-div"><i class="el-icon el-icon-status-SIM-avaliable1"></i> <%=rb.getString("WeiFenPei")%></div>
                        <div v-if="scope.row.assignStatus == '1'" class="active-status-div"><i class="el-icon el-icon-status-SIM-inuse success-status"></i> <%=rb.getString("YiFenPei")%></div>
                    </template>
                </el-table-column>
                <el-table-column v-if="false" label="<%=rb.getString("TongBuJieGuo")%>" prop="syncResult">
                    <template slot-scope="scope">
                        <div v-if="scope.row.syncResult == '2'" class="active-status-div">
                            <img src="${ctx}/css/images/main/monitor-ico/monitor-syning.gif" style="margin-right: 10px;"/> <%=rb.getString("ZhengZaiTongBu")%>
                        </div>
                        <div v-if="scope.row.syncResult == '1'" class="active-status-div">
                            <img src="${ctx}/css/images/main/monitor-ico/tnsuccessomc.png" style="margin-right: 10px;"/> <%=rb.getString("TongBuChengGong")%>
                        </div>
                        <div v-if="scope.row.syncResult == '0'" class="active-status-div">
                            <img src="${ctx}/css/images/main/monitor-ico/monitor-nosynomc.png" style="margin-right: 10px;"/> <%=rb.getString("TongBuShiBai")%>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column v-if="false" label="<%=rb.getString("TongBuShiJian")%>" prop="syncTime"></el-table-column>
                <el-table-column label="APN" prop="apnInfos">
                    <template slot-scope="scope">
                        <el-popover v-if="scope.row.apnInfos && scope.row.apnInfos.length">
                            <span slot="reference">
                                {{scope.row.apnInfos[0].apnName}}
                                <span v-if="scope.row.apnInfos.length>1">
                                    [ <span style="color:#4d84ff;">{{scope.row.apnInfos.length}}</span>
                                    <i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;color:#4d84ff;"></i>]
                                </span>
                            </span>
                            <div class="apn-list">
                                <div v-for="(item,index) in scope.row.apnInfos">
                                    APN{{item.ContextId}} APN Name:  {{item.apnName}}
                                </div>
                            </div>
                        </el-popover>
                    </template>
            </el-table-column>
                <el-table-column label="UE AMBR UL" prop="ue_ambr_ul"></el-table-column>
                <el-table-column label="UE AMBR DL" prop="ue_ambr_dl"></el-table-column>
            </el-ctable>
        </div>
        <div class="imsiAllocatedItemCls">
            <div class="apnAndMisisdBoxCls">
                <div>
                    <el-ctable ref="apnlist" type="card"
                        :pagination="true"
                        :url="apnUrl"
                        :query-params="apnParams"
                        :card-option="cardOption">
                        <template slot="toolbar">
                            <div class="toolbarHeadBoxCls">
                                <div style="display: flex;align-items: center;">
                                    <h3 style="padding-left: 10px;">APN List</h3>
                                    <el-query type="normal" placeholder="APN Name" @query="queryApn"></el-query>
                                </div>
                                <div class="toolbarHeadBtnCls">
                                    <div v-if="isWritable" class="newIconBoxCls-bt" @click="syncApn" tip="<%=rb.getString("TongBu")%>">
                                        <span @click="syncApn" class="el-icon el-icon-circle-sync"></span>
                                    </div>
                                </div>
                            </div>
                        </template>
                    </el-ctable>
                </div>
                <div>
                    <el-ctable ref="msisdnlist" class="no-border-tb"
                        id="msisdnTableList"
                        :url="msisdnUrl"
                        :time="6"
                        :query-params="msisdnParams"
                    >
                        <template slot="toolbar">
                            <div class="toolbarHeadBoxCls">
                                <div style="display: flex;align-items: center;">
                                    <h3 style="padding-left: 10px;">MSISDN</h3>
                                    <el-query type="normal" placeholder="MSISDN / IMSI" @query="queryMsisdn"></el-query>
                                </div>
                                <div class="toolbarHeadBtnCls">
                                    <div v-if="isWritable" class="newIconBoxCls-bt" @click="syncMsisdn" tip="<%=rb.getString("TongBu")%>">
                                        <span class="el-icon el-icon-circle-sync" style="font-size: 16px;"></span>
                                    </div>
                                    <div class="newIconBoxCls-bt" @click="exportMSISDN" tip="<%=rb.getString("DaoChu")%>">
                                        <span class="el-icon el-icon-circle-export" style="font-size: 16px;"></span>
                                    </div>
                                </div>
                            </div>
                        </template>
                        <el-table-column label="MSISDN" prop="msisdn"></el-table-column></el-table-column>
                        <el-table-column label="IMSI" prop="imsi"></el-table-column></el-table-column>
                        <el-table-column label='<%=rb.getString("BangDingZhuangTai")%>' prop="activeStaus">
                            <template slot-scope="scope">
                                <div style="display: flex;align-items: center;">
                                    <el-tag size="mini" type="success" v-if="scope.row.activeStatus == '1'"><%=rb.getString("BangDing")%></el-tag>
                                    <el-tag size="mini" type="danger" v-else><%=rb.getString("WeiBangDing")%></el-tag>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column label="Assignment" prop="assignStatus">
                            <template slot-scope="scope">
                                <div style="display: flex;align-items: center;">
                                    <span v-if="scope.row.assignStatus == '1'">
                                        <i class="el-icon el-icon-status-vouchercard-inuse" style="color: #67c23a;"></i> <%=rb.getString("ShiYongZhong")%>
                                    </span>
                                    <span v-else>
                                        <i class="el-icon el-icon-status-vouchercard-available" style="color: #4D84FF;"></i> <%=rb.getString("WeiFenPei")%>
                                    </span>
                                </div>
                            </template>
                        </el-table-column>
                    </el-ctable>
                </div>
            </div>
        </div>
    </div>
</div>

<script>
    new Vue({
        el: '#imsi_allocated_ctn',
        data() {

            return {
                imsiURL: '${ctx}/cell/imsi/queryImsiPageList.action',
                imsiParams: {
                    timeZone: timeZone,
                    searchText: ''
                },

                apnUrl: '${ctx}/epc/apnconfig/getGwApnInfos.action',
                apnParams: {
                    searchText: '',
                    timeZone: timeZone
                },

                msisdnUrl: '${ctx}/cell/imsi/queryMsisdnInfoPageList.action',
                msisdnParams: {
                    timeZone: timeZone,
                    searchText: ''
                },

                cardOption: {
                    fields: [
                        {label: '<%=rb.getString("APNMingCheng")%>', field: 'APN_NAME'},
                        {label: 'APN Uplink Limit', field: 'APN_AMBR_UL',formatter: function(mRow, row){
                            var str = row.APN_AMBR_UL?row.APN_AMBR_UL+'Mbps':'';

                            return str;
                        }},
                        {label: 'APN Downlink Limit', field: 'APN_AMBR_DL',formatter: function(mRow, row){
                            var str = row.APN_AMBR_DL?row.APN_AMBR_DL+'Mbps':'';

                            return str;
                        }},
                        {label: 'Primary DNS IP', field: 'PRIMARY_DNS_IPADDR'},
                        {label: 'Secondary DNS IP', field: 'SECONDARY_DNS_IPADDR'},
                        {label: 'QCI', field: 'QCI'},
                        {label: 'ARP Priority Level', field: 'ARP_PRIORITYLEVEL'},
                        {label: 'ARP PCI', field: 'ARP_PCI', formatter: function(mRow, row){
                            return row.ARP_PCI == '1'?'Open':'Close';
                        }},
                        {label: 'ARP PVI', field: 'ARP_PVI', formatter: function(mRow, row){
                            return row.ARP_PVI == '1'?'Open':'Close';
                        }},
                        /*{label: 'IPv4', field: 'IPPOOL_INFO',formatter: function(mRow, row){
                            var list = [],
                                str = '';

                            if(row.IPPOOL_INFO) {
                                row.IPPOOL_INFO.map(function(item){
                                    list.push(item['START_SERVED_PARTY_IPV4_ADDRESS'] + ' ~ ' + item['END_SERVED_PARTY_IPV4_ADDRESS']);
                                })
                            }

                            if(row.IPPOOL_INFO && row.IPPOOL_INFO.length) {
                                str = '<div title="'+list.join('\n')+'">';

                                var item = row.IPPOOL_INFO[0];
                                str += '<div>' + item['START_SERVED_PARTY_IPV4_ADDRESS'] + ' ~ ' + item['END_SERVED_PARTY_IPV4_ADDRESS'] + '</div>';

                                if(row.IPPOOL_INFO.length>1) {
                                    str += '<span style="color: blue;"> (' +row.IPPOOL_INFO.length+ ')</span>';
                                }

                                str += '</div>';
                            }

                            return str;
                        }}*/
                    ]
                }
            };
        },
        computed: {
            isWritable() {

            	return writableMap.CODE_ENB_DEVICE_HALOB == true;
            }
        },
        methods: {
            sortList(list) {
                return list.sort(function(n, m){
                    return n.ContextId - m.ContextId;
                });
            },
            queryIMSI(text) {
                this.imsiParams.searchText = text;
            },
            queryApn(txt) {
                this.apnParams.searchText = txt;
            },
            queryMsisdn(txt) {
                this.msisdnParams.searchText = txt;
            },
            exportIMSI() {
                var vm = this;

                exportByForm('${ctx}/cell/imsi/exportImsiList.action',vm.imsiParams);
            },
            exportMSISDN() {
                var vm = this;

                exportByForm('${ctx}/cell/imsi/exportMsisdnList.action',vm.msisdnParams);
            },
            syncApn() {
                var vm = this,
                    url = '${ctx}/epc/apnconfig/syncApnInfo.action';

                axios.post(url).then(function(res){
                    var data = res.data;

                    if(data.success == true) {
                        eventBus.$message({
                            message: '<%=rb.getString("ChengGong")%>',
                            type: 'success'
                        });

                        vm.$refs.apnlist.refresh();
                    }else {
                        vm.$message({
                            message: data.message,
                            type: 'error'
                        });
                    }
                });
            },
            syncMsisdn() {
                var vm = this,
                    url = '${ctx}/cell/imsi/syncMsisdnInfo.action';

                axios.post(url).then(function(res){
                    var data = res.data;

                    if(data.success == true) {
                        eventBus.$message({
                            message: '<%=rb.getString("ChengGong")%>',
                            type: 'success'
                        });

                        vm.$refs.msisdnlist.refresh();
                    }else {
                        vm.$message({
                            message: data.message,
                            type: 'error'
                        });
                    }
                });
            },
            closeSlide() {
                eventBus.$emit('hide-imsi');
            }
        }
    })

</script>