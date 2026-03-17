<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwInformationQueryPage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egwInformationQueryPage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egwInformationQueryPage .itemMainBoxTitle {
	height:50px;
    line-height: 50px;
	font-size:14px;
	font-weight:bold;
    position: relative;
}
#egwInformationQueryPage .linkStatusContent{
    height: 100%;
    padding: 0px 20px 20px;
    overflow: auto;
}
#egwInformationQueryPage .linkStatusItemBoxCls{
	position: relative;
}
#egwInformationQueryPage .linkStatusTableBoxCls{
    height: 300px;
    overflow: hidden;
    box-sizing: border-box;
    border:1px solid #d5dcec;
    border-radius: 4px;
}
#egwInformationQueryPage .activeStatusItem .el-icon,#egwInformationQueryPage .inactiveStatusItem .el-icon{
    font-size:20px;
    vertical-align:bottom;
    margin-right:5px;
}
#egwInformationQueryPage .activeStatusItem .el-icon-status-active:before{
    color:#67D972;
}
#egwInformationQueryPage .inactiveStatusItem .el-icon-status-active:before{
    color:#E88282;
}
#egwInformationQueryPage .toolbarBoxCls{
    padding: 10px 0px;
}
#egwInformationQueryPage .egwTabPaneContent{
    margin: 35px 0px 0px 40px;
}
#egwInformationQueryPage .egwTabPaneContent .egwTabPaneItemCls{
    margin-bottom: 20px;
    font-size: 14px;
    display: flex;
}
#egwInformationQueryPage .egwTabPaneContent .egwTabPaneItemCls .egwTabPaneItemLabelCls{
    color: #7A7992;
    width: 360px;
}
#egwInformationQueryPage .egwTabPaneContent .egwTabPaneItemCls .egwTabPaneItemValueCls{
    color: rgba(0, 0, 0, 0.8);
}
#egwInformationQueryPage .querySigGWStatisticBox{
	margin-bottom:15px;
	display: flex;
	align-items: center;
}
#egwInformationQueryPage .querySigGWStatisticBox .el-input.el-input--small{
	width: 200px;
}
#egwInformationQueryPage .sigGWStatisticMainBox{
	position: relative;
}
#egwInformationQueryPage .sigGWStatisticItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	margin-bottom: 10px;
}
#egwInformationQueryPage .sigGWStatisticItemBoxCls >div{
	flex:1;
    display: flex;
}
#egwInformationQueryPage .paramItemLabel{
	color:#7a7992;
}
#egwInformationQueryPage .paramItemValue{
	height: 18px;
}
#egwInformationQueryPage .isZhWidth{
    width: 150px;
}
#egwInformationQueryPage .isEnWidth{
    width: 280px;
}
</style>

<div id="egwInformationQueryPage">
    <div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:20px;top:5px;" @click="syncSubmit" tip='<%=rb.getString("TongBu")%>'>
        <span class="el-icon el-icon-circle-refresh"></span>
    </div>	
	<div class="itemMainBoxCls">
		<el-tabs class="fit newTabs" v-model="activeName" style='height:100%' @tab-click="tabClick">
            <el-tab-pane label="SeGW" name="SeGW">
                <div class="egwTabPaneContent">
                    <div class="egwTabPaneItemCls">
                        <span style="display:inline-block;width:150px;">IKE SA</span> 
                        <span>{{segwData.ikeSA}}</span> 
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane v-if="showSigGW4G" label="4G SigGW" name="4GSigGW">
                <div class="linkStatusContent">
                    <div class="linkStatusItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            4G <%=rb.getString("SigGWDeJiZhanShiShiLiuLiangChaXun")%>
                        </div>
                        <div class="querySigGWStatisticBox">
                            <span style="margin-right:10px;"><%=rb.getString("JiZhanLiDu")%></span>
                            <el-query type="normal" @query="query4GSigGWStatistic" placeholder="ECI"></el-query>
                        </div>
                        <div class="sigGWStatisticMainBox" v-loading="query4GSigGWStatisticLoading">
							<div class="sigGWStatisticItemBoxCls">
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("ShouBaoZongLiang")%>（packets）</div>
									<div class="paramItemValue">{{sigGWStatisticData_4G.downlinkPackets}}</div>
								</div>
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("FaBaoZongLiang")%>（packets）</div>
									<div class="paramItemValue">{{sigGWStatisticData_4G.uplinkPackets}}</div>
								</div>
							</div>
                            <div class="sigGWStatisticItemBoxCls">
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("JieShouZongZiJieShu")%>（KBytes）</div>
									<div class="paramItemValue">{{sigGWStatisticData_4G.downlinkKBytes}}</div>
								</div>
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("FaSongZongZiJieShu")%>（KBytes）</div>
									<div class="paramItemValue">{{sigGWStatisticData_4G.uplinkKBytes}}</div>
								</div>
							</div>
                        </div>
                    </div>
                    <div class="linkStatusItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            <%=rb.getString("ZongLianLuQingKuang")%> HeNB-WCG
                            <span class="el-icon el-icon-operation-export" style="position:absolute;right:0px;top:17px;" @click="exportLinkTable('4G_ENB')"></span>
                        </div>
                        <div class="linkStatusTableBoxCls">
                            <el-ctable 
                                ref="egwAndEnb4GTable" 
                                :rownumber="true" 
                                id="egwAndEnb4GTable" 
                                :url="egwAndEnb4GTableUrl"
                                :query-params="queryEgwAndEnb4GParams"
                                height="100%" 
                                pagination="true"
                            >
                                <template slot="toolbar">
                                    <div class="toolbarBoxCls">
                                        <el-query ref="egwAndEnb4GTableQuery" type="normal" @query="queryEgwAndEnb4G" placeholder="<%=rb.getString("EnodebId")%>"></el-query>
                                    </div>
                                </template>
                                <el-table-column label='<%=rb.getString("EnodebId")%>' min-width="130" prop="EnbId"></el-table-column>
                                <el-table-column label='<%=rb.getString("CellID")%>' min-width="120" prop="CellId" show-overflow-tooltip></el-table-column>
                                <el-table-column label='IP' min-width="120" prop="EnbIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("DuanKou")%>' min-width="100" prop="EnbPort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("JieRuShiJianTiShi")%>' min-width="150" prop="AccessTime" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="170" prop="Status" show-overflow-tooltip>
                                    <template slot-scope="scope">
                                        <div class='activeStatusItem' v-if="scope.row.Status == 'Active'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("JiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.Status == 'Inactive'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("QuJiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.Status == 'Unknown'">
                                            {{scope.row.Status}}
                                        </div>
                                    </template>
                                </el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                    <div class="linkStatusItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            <%=rb.getString("ZongLianLuQingKuang")%> WCG-MME
                            <span class="el-icon el-icon-operation-export" style="position:absolute;right:0px;top:17px;" @click="exportLinkTable('4G_MML')"></span>
                        </div>
                        <div class="linkStatusTableBoxCls">
                            <el-ctable 
                                ref="egwAndMme4GTable" 
                                :rownumber="true" 
                                id="egwAndMme4GTable" 
                                :url="egwAndMme4GTableUrl"
                                :query-params="queryEgwAndMme4GParams" 
                                height="100%"
                                pagination="true"
                            >
                                <el-table-column label='<%=rb.getString("LianLuSuoYin")%>' min-width="100" prop="LinkIndex"></el-table-column>
                                <el-table-column label='<%=rb.getString("EnodebId")%>' min-width="100" prop="EnbId"></el-table-column>
                                <el-table-column label='<%=rb.getString("BenDuanIP")%>' min-width="120" prop="LocalIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("BenDuanIP2")%>' min-width="120" prop="SecondLocalIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("BenDuanDuanKou")%>' min-width="120" prop="LocalPort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("DuiDuanIP")%>' min-width="120" prop="RemoteIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("DuiDuanDuanKou")%>' min-width="120" prop="RemotePort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("S1APZhuangTai")%>' min-width="160" prop="S1apStatus" show-overflow-tooltip>
                                    <template slot-scope="scope">
                                        <div class='activeStatusItem' v-if="scope.row.S1apStatus == 'Active'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("JiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.S1apStatus == 'Inactive'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("QuJiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.S1apStatus == 'Unknown'">
                                            {{scope.row.S1apStatus}}
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("SCTPZhuangTai")%>' min-width="120" prop="SctpStatus"></el-table-column>
                                <el-table-column label='<%=rb.getString("MMENengLiZhi")%>' min-width="120" prop="MmeCapacity"></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane v-if="showSigGW5G" label="5G SigGW" name="5GSigGW">
                <div class="linkStatusContent">
                    <div class="linkStatusItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            5G <%=rb.getString("SigGWDeJiZhanShiShiLiuLiangChaXun")%>
                        </div>
                        <div class="querySigGWStatisticBox">
                            <span style="margin-right:10px;"><%=rb.getString("JiZhanLiDu")%></span>
                            <el-query type="normal" @query="query5GSigGWStatistic" placeholder="ECI"></el-query>
                        </div>
                        <div class="sigGWStatisticMainBox" v-loading="query5GSigGWStatisticLoading">
							<div class="sigGWStatisticItemBoxCls">
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("ShouBaoZongLiang")%>（packets）</div>
									<div class="paramItemValue">{{sigGWStatisticData_5G.downlinkPackets}}</div>
								</div>
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("FaBaoZongLiang")%>（packets）</div>
									<div class="paramItemValue">{{sigGWStatisticData_5G.uplinkPackets}}</div>
								</div>
							</div>
                            <div class="sigGWStatisticItemBoxCls">
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("JieShouZongZiJieShu")%>（KBytes）</div>
									<div class="paramItemValue">{{sigGWStatisticData_5G.downlinkKBytes}}</div>
								</div>
								<div>
									<div :class="isZH ? 'paramItemLabel isZhWidth' : 'paramItemLabel isEnWidth'"><%=rb.getString("FaSongZongZiJieShu")%>（KBytes）</div>
									<div class="paramItemValue">{{sigGWStatisticData_5G.uplinkKBytes}}</div>
								</div>
							</div>
                        </div>
                    </div>
                    <div class="linkStatusItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            <%=rb.getString("ZongLianLuQingKuang")%> HgNB-WCG
                            <span class="el-icon el-icon-operation-export" style="position:absolute;right:0px;top:17px;" @click="exportLinkTable('5G_ENB')"></span>
                        </div>
                        <div class="linkStatusTableBoxCls">
                            <el-ctable 
                                ref="egwAndEnb5GTable" 
                                :rownumber="true" 
                                id="egwAndEnb5GTable" 
                                :url="egwAndEnb5GTableUrl"
                                :query-params="queryEgwAndEnb5GParams"
                                height="100%" 
                                pagination="true"
                            >
                                <template slot="toolbar">
                                    <div class="toolbarBoxCls">
                                         <el-query ref="egwAndEnb5GTableQuery" type="normal" @query="queryEgwAndEnb5G" placeholder="<%=rb.getString("GnodebId")%>"></el-query>
                                    </div>
                                </template>
                                <el-table-column label='<%=rb.getString("GnodebId")%>' min-width="130" prop="EnbId"></el-table-column>
                                <el-table-column label='<%=rb.getString("CellID")%>' min-width="120" prop="CellId" show-overflow-tooltip></el-table-column>
                                <el-table-column label='IP' min-width="120" prop="EnbIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("DuanKou")%>' min-width="100" prop="EnbPort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("JieRuShiJianTiShi")%>' min-width="150" prop="AccessTime" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="170" prop="Status" show-overflow-tooltip>
                                    <template slot-scope="scope">
                                        <div class='activeStatusItem' v-if="scope.row.Status == 'Active'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("JiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.Status == 'Inactive'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("QuJiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.Status == 'Unknown'">
                                            {{scope.row.Status}}
                                        </div>
                                    </template>
                                </el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                    <div class="linkStatusItemBoxCls">
                        <div class="itemMainBoxTitle">
                            <span class="el-icon el-icon-splitGroup"></span>
                            <%=rb.getString("ZongLianLuQingKuang")%> WCG-AMF
                            <span class="el-icon el-icon-operation-export" style="position:absolute;right:0px;top:17px;" @click="exportLinkTable('5G_MML')"></span>
                        </div>
                        <div class="linkStatusTableBoxCls">
                            <el-ctable 
                                ref="egwAndMme5GTable" 
                                :rownumber="true" 
                                id="egwAndMme5GTable" 
                                :url="egwAndMme5GTableUrl"
                                :query-params="queryEgwAndMme5GParams" 
                                height="100%"
                                pagination="true"
                            >
                                <el-table-column label='<%=rb.getString("LianLuSuoYin")%>' min-width="100" prop="LinkIndex"></el-table-column>
                                <el-table-column label='<%=rb.getString("GnodebId")%>' min-width="100" prop="EnbId"></el-table-column>
                                <el-table-column label='<%=rb.getString("BenDuanIP")%>' min-width="120" prop="LocalIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("BenDuanIP2")%>' min-width="120" prop="SecondLocalIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("BenDuanDuanKou")%>' min-width="120" prop="LocalPort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("DuiDuanIP")%>' min-width="120" prop="RemoteIp" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("DuiDuanDuanKou")%>' min-width="120" prop="RemotePort" show-overflow-tooltip></el-table-column>
                                <el-table-column label='<%=rb.getString("S1APZhuangTai")%>' min-width="160" prop="S1apStatus" show-overflow-tooltip>
                                    <template slot-scope="scope">
                                        <div class='activeStatusItem' v-if="scope.row.S1apStatus == 'Active'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("JiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.S1apStatus == 'Inactive'">
                                            <span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span><%= rb.getString("QuJiHuo")%>
                                        </div>
                                        <div class='inactiveStatusItem' v-show="scope.row.S1apStatus == 'Unknown'">
                                            {{scope.row.S1apStatus}}
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("SCTPZhuangTai")%>' min-width="120" prop="SctpStatus"></el-table-column>
                                <el-table-column label='<%=rb.getString("AMFNengLiZhi")%>' min-width="120" prop="MmeCapacity"></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
		</el-tabs>
	</div>
</div>

<script>
var egwInformationQueryPage = new Vue({
	el: '#egwInformationQueryPage', 
	data() {
		var vm = this;
		return {
            egwCode:'',
            egwSn:'',
            rowData:{
                generation:'',
            },
            activeName:'SeGW',
            segwData:{
                 ikeSA:'',
            },
            queryEgwAndEnb4GParams:{
                timeZone:timeZone,
                eci:''
            },
            queryEgwAndMme4GParams:{
                timeZone:timeZone,
            },
            queryEgwAndEnb5GParams:{
                timeZone:timeZone,
                 eci:''
            },
            queryEgwAndMme5GParams:{
                timeZone:timeZone,
               
            },
            egwAndEnb4GTableUrl:'',
            egwAndMme4GTableUrl:'',
			egwAndEnb5GTableUrl:'',
            egwAndMme5GTableUrl:'',

            sigGWStatisticData_4G:{
				uplinkPackets:'',
				downlinkPackets:'',
				uplinkKBytes:'',
				downlinkKBytes:'',
			},
			sigGWStatisticData_5G:{
				uplinkPackets:'',
				downlinkPackets:'',
				uplinkKBytes:'',
				downlinkKBytes:'',
			},
            query4GSigGWStatisticLoading:false,
			query5GSigGWStatisticLoading:false,
		};
	},
	computed: {
        showSigGW4G(){
			return this.rowData.generation == 'SigGW4G' || this.rowData.generation == 'SigGW4/5G' || this.rowData.generation == 'SigGW4G+SeGW' || this.rowData.generation == 'SigGW4/5G+SeGW' 
		},
		showSigGW5G(){
			return this.rowData.generation == 'SigGW5G' || this.rowData.generation == 'SigGW4/5G' || this.rowData.generation == 'SigGW5G+SeGW' || this.rowData.generation == 'SigGW4/5G+SeGW' 
		},
        isZH() {
			return isLocalZH == true;
		},
        optBtnShow() {
			return writableMap['CODE_EGW'] == true;
		},
    },
	methods: {
        // 初始化
		init(row,code,sn,status){
		    var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;
            vm.rowData = row;
            vm.queryEgwAndEnb4GParams.egwCode = code;
            vm.queryEgwAndMme4GParams.egwCode = code;
            vm.queryEgwAndEnb5GParams.egwCode = code;
            vm.queryEgwAndMme5GParams.egwCode = code;
            vm.getSeGWInformationData();
		},
        // Tab 切换
		tabClick(val){
            var vm = this,
                str = Math.random().toString();
            if(vm.activeName == 'SeGW'){
                vm.getSeGWInformationData();
            }else if(vm.activeName == '4GSigGW'){
                vm.egwAndEnb4GTableUrl = '${ctx}/egw/monitor/getEgwENBList.action?randomCode='+ str;
                vm.egwAndMme4GTableUrl = '${ctx}/egw/monitor/getEgwMMEList.action?randomCode='+ str;
            }else if(vm.activeName == '5GSigGW'){
                vm.egwAndEnb5GTableUrl = '${ctx}/egw/monitor/getEgwGNBList.action?randomCode='+ str;
                vm.egwAndMme5GTableUrl = '${ctx}/egw/monitor/getEgwGNBMMEList.action?randomCode='+ str;
            }
        },
        getSeGWInformationData(){
            var vm = this,
                params = {
                   egwCode: vm.egwCode
                };
            axios.post("${ctx}/egw/config/getIkeSA.action",stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    vm.segwData.ikeSA = data.ikeSA ? data.ikeSA : '';
                }
            });
        },
        // queryEgwAndEnb4G 表格查询
		queryEgwAndEnb4G(val){
            var vm =this;
            vm.queryEgwAndEnb4GParams.eci = val;
        },
        // queryEgwAndEnb5G 表格查询
		queryEgwAndEnb5G(val){
            var vm =this;
            vm.queryEgwAndEnb5GParams.eci = val;
        },
        // 导出总链路表格
        exportLinkTable(type){
            var vm = this,
                urls = ''
                params = {
                    timeZone: timeZone,
                    egwCode: vm.egwCode
                };
            if(type == '4G_ENB'){
                urls = '${ctx}/egw/monitor/exportEgwENBList.action';
                params.eci = vm.queryEgwAndEnb4GParams.eci;
            }else if(type == '4G_MML'){
                urls = '${ctx}/egw/monitor/exportEgwMMEList.action';
            }else if(type == '5G_ENB'){
                urls = '${ctx}/egw/monitor/exportEgwGNBList.action';
            }else if(type == '5G_MML'){
                urls = '${ctx}/egw/monitor/exportEgwGNBMMEList.action';
                params.eci = vm.queryEgwAndEnb5GParams.eci;
            }
            exportByForm(urls,params);
        },
        // 同步
        syncSubmit(){
            var vm = this,
                urls='${ctx}/egw/config/refreshConfig.action',
                params={
                    egwCode: vm.egwCode,
                    refreshType: 'INFORMATIONQUERY'
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("MingLingYiXiaFa")%>',
                        type:'success',
                    });
                }else{
                    vm.$message.error(data["message"])
                }
            })
        },
        // 4G SigGW的基站粒度统计查询
		query4GSigGWStatistic(val){
			var vm = this,
                params = {
                   egwCode: vm.egwCode,
				   eci:val,
				   type:'4G'
                };
			vm.query4GSigGWStatisticLoading = true;
            axios.post("${ctx}/egw/monitor/getSigGWECIPakets.action",stringify(params)).then(function(response){
                var data = response.data;
                if(data){
					Object.keys(vm.sigGWStatisticData_4G).map((key)=>{
						vm.sigGWStatisticData_4G[key] = data[key] ? data[key] : '<%=rb.getString("MeiShuJu")%>';
					})
					vm.query4GSigGWStatisticLoading = false;
                }
            });

		},
		// 5G SigGW的基站粒度统计查询
		query5GSigGWStatistic(val){
			var vm = this,
                params = {
                   egwCode: vm.egwCode,
				   eci:val,
				   type:'5G'
                };
			vm.query5GSigGWStatisticLoading = true;
            axios.post("${ctx}/egw/monitor/getSigGWECIPakets.action",stringify(params)).then(function(response){
                var data = response.data;
                if(data){
					Object.keys(vm.sigGWStatisticData_5G).map((key)=>{
						vm.sigGWStatisticData_5G[key] = data[key] ? data[key] : '<%=rb.getString("MeiShuJu")%>';
					})
					vm.query5GSigGWStatisticLoading = false;
                }
            });

		},
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
