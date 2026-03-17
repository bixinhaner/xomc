<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwLinkStatusPage .linkStatusContent{
		display: flex;
		justify-content: space-around;
		height: 90%;
		margin-top: 20px
	}
	#egwLinkStatusPage .linkStatusContent .contentLeftBoxCls{
		width: 49%;
		height: 100%;
		border:1px solid #E9E9E9;
		border-top: 2px solid #4D84FF; 
		box-sizing:border-box; 
	}
	#egwLinkStatusPage .linkStatusContent .contentRightBoxCls{
		width: 49%;
		height: 100%;
		border:1px solid #E9E9E9;
		border-top: 2px solid #4D84FF; 
		box-sizing:border-box; 
	}
	#egwLinkStatusPage .linkStatusContent .tableTitleCls{
		position: relative;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 12px;
		color: #333333;
		font-weight: bold;
		padding: 0 10px;
		
	}
	#egwLinkStatusPage .tableTitleCls .el-icon::before{
		font-size: 24px;
	}
	#egwLinkStatusPage .linkStatusContent .tableBoxCls{
		height: calc(100% - 40px) !important;
	}
	.activeStatusItem .el-icon,.inactiveStatusItem .el-icon{
		font-size:20px;
		vertical-align:bottom;
		margin-right:5px;
	}
	.activeStatusItem .el-icon-status-active:before{
		color:#67D972;
	}
	.inactiveStatusItem .el-icon-status-active:before{
		color:#E88282;
	}
</style>
<div class="panelDefault" id="egwLinkStatusPage" style="overflow:hidden">
	<div class="linkStatusContent">
		<div class="contentLeftBoxCls">
			<div class="tableTitleCls">
				<span><%=rb.getString("ZongLianLuQingKuang")%> eGW-eNB</span>
				<span class="el-icon el-icon-circle-export" @click="exportEgwEnbTable"></span>
			</div>
			<div class="tableBoxCls">
				<el-ctable 
					ref="egwAndEnbTable" 
					:rownumber="true" 
					id="egwAndEnbTable" 
					:url="egwAndEnbTableUrl"
					:query-params="queryParams"
					height="100%" 
					@load-success="tableLoadSuccess('egwAndEnb')" 
					pagination="true"
				>
					<el-table-column label='<%=rb.getString("EnodebId")%>' min-width="130" prop="EnbId"></el-table-column>
					<el-table-column label='CellID' min-width="120" prop="CellId" show-overflow-tooltip></el-table-column>
					<el-table-column label='IP' min-width="120" prop="EnbIp" show-overflow-tooltip></el-table-column>
					<el-table-column label='Port' min-width="100" prop="EnbPort" show-overflow-tooltip></el-table-column>
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
		<div class="contentRightBoxCls">
			<div class="tableTitleCls">
				<span><%=rb.getString("ZongLianLuQingKuang")%> eGW-MME</span>
				<span class="el-icon el-icon-circle-export" @click="exportEgwMMETable"></span>
			</div>
			<div class="tableBoxCls">
				<el-ctable 
					ref="egwAndMmeTable" 
					:rownumber="true" 
					id="egwAndMmeTable" 
					:url="egwAndMmeTableUrl"
					:query-params="queryParams" 
					height="100%"
					@load-success="tableLoadSuccess('egwAndMme')" 
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
</div>

<script type="text/javascript">
	
	var egwLinkStatusVue = new Vue({
		el: '#egwLinkStatusPage',
		data(){
			var vm = this;
			return {
				data:[],
				queryParams:{
					timeZone:timeZone,
				},
				egwAndEnbTableUrl:'',
				egwAndMmeTableUrl:'',
				tableStatus:{
					egwAndEnb:{loaded:false},
					egwAndMme:{loaded:false}
				}
			}
		},
		computed: {
		
		},
		watch: {},
		methods: {
			// 初始化
			init(data){
				var vm = this;
				vm.egwCode = data.egwCode;
				vm.egwAndEnbTableUrl = '${ctx}/egw/monitor/getEgwENBList.action?egwCode='+vm.egwCode;
				vm.egwAndMmeTableUrl = '${ctx}/egw/monitor/getEgwMMEList.action?egwCode='+vm.egwCode;
				
			},
			// 导出总链路情况egw-enb表格
			exportEgwEnbTable(){
				var vm = this;
				exportByForm("${ctx}/egw/monitor/exportEgwENBList.action",{
					timeZone: timeZone,
					egwCode: vm.egwCode
				});
			},
			// 导出总链路情况egw-MME表格
			exportEgwMMETable(){
				var vm = this;
				exportByForm("${ctx}/egw/monitor/exportEgwMMEList.action",{
					timeZone: timeZone,
					egwCode: vm.egwCode
				});
			},
			// 表格加载成功回调
			tableLoadSuccess(val){
				var vm = this;
				
				vm.tableStatus[val].loaded = true;
				if(vm.tableStatus.egwAndEnb.loaded == true && vm.tableStatus.egwAndMme.loaded == true){
					egwMonitor.settingslideCls = '';
				}
			},
		},
		created(){},
		mounted(){
			eventBus.$off('link-init').$on('link-init',this.init);
		}
	});
</script>