<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.alarmMinor .el-icon:before{
		color: #FFDA41;
		font-size: 20px;
	}
	.alarmMajor .el-icon:before{
		color: #FF973E;
		font-size: 20px;
	}
	.alarmCritical .el-icon:before{
		color: #FC5959;
		font-size: 20px;
	}
	.alarmWarning .el-icon:before{
		color: #60BEFC;
		font-size: 20px;
	}
	.el-tooltip__popper{
		max-width: 800px;
	}
	.severityCls >div{
		height: 28px;
		width: 140px;
		display: flex;
		padding-left: 10px;
		align-items: center;
		cursor: pointer;
	}
	.severityPopperCls{
		padding: 0px!important;
	}
	.severityCls >div:hover{
		background-color: #E9E9E9;
	}
	.no-outline:focus {
		outline: none ;
	}
</style>
<div class="pageDefault" id='alarmLibInfo'>
	<div class="container">
        <el-tabs class="fit">
          <el-tab-pane label="<%=rb.getString("AlarmlevelConfig")%>">
            <!-- 操作按钮 -->
            <div class="operations">
              <div class="placeholder-bt circleIcon" placeholder="<%=rb.getString("DaoChu")%>">
                <span class="el-icon el-icon-circle-export" @click="exportAlarmLib"></span>
              </div>
            </div>
            <!-- 表格组件 -->
            <el-ctable
                :url="libraryTableUrl" 
                :query-params="params"
				id="libraryTable"
                ref="libraryTable" 
				:height="height" 
				:page-size="pageSize" 
				:page-list="pageList" 
				pagination="true">
                	<!-- 列表toolbar -->
                <template slot="toolbar">
                    <el-query type="normal" @query="queryLibrary" placeholder="<%=rb.getString("QingShuRuGaoJingBiaoShiMaORGaoJingYuanYin")%>"></el-query>
                </template>
                	<!-- 列表columns -->
                <el-table-column prop="DEVICE_TYPE_NAME" label="<%=rb.getString("XinGaoJingYuan")%>" min-width="110" sortable></el-table-column>
                <el-table-column prop="ALARM_IDENTIFIER" label="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>" min-width="150" sortable></el-table-column>
                <el-table-column prop="ALARM_NAME" label="<%=rb.getString("KeNengYuanYin")%>" min-width="300"></el-table-column>
				<el-table-column prop="SERVERITY_TYPE" label="<%=rb.getString("YanZhongChengDu")%>" min-width="140" sortable>
					<template slot-scope="scope">
						<el-popover placement="bottom" width="150" trigger="click" popper-class="severityPopperCls">
							<div class="severityCls">
								<div class="alarmMinor" @click="resetAlarmServerity('31003',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("CiYaoGaoJing")%></div>
								<div class="alarmMajor" @click="resetAlarmServerity('31002',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("ZhuYaoGaoJing")%></div>
								<div class="alarmCritical" @click="resetAlarmServerity('31001',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("JinJiGaoJing")%></div>
								<div class="alarmWarning" @click="resetAlarmServerity('31004',scope.row.ALARM_IDENTIFIER)"><span class="el-icon el-icon-status-alarm" style="margin-right:20px;"></span><%=rb.getString("JingGaoGaoJing")%></div>
							</div>
							<div v-if="scope.row.SERVERITY_TYPE == 'Minor'" class="alarmMinor no-outline" style="cursor: pointer;" slot="reference">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
							</div>
							<div v-else-if="scope.row.SERVERITY_TYPE == 'Major'"  class="alarmMajor no-outline" style="cursor: pointer;" slot="reference">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
							</div>
							<div v-else-if="scope.row.SERVERITY_TYPE == 'Critical'" class="alarmCritical no-outline" style="cursor: pointer;" slot="reference">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
							</div>
							<div v-else-if="scope.row.SERVERITY_TYPE == 'Warning'" class="alarmWarning no-outline" style="cursor: pointer;" slot="reference">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
							</div>

						</el-popover>
						
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="160" prop="EVENT_TYPE" show-overflow-tooltip>
						<template slot-scope="scope">
							<div v-if="scope.row.EVENT_TYPE == '30000'">
								<span><%=rb.getString("TongXinGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30001'">
								<span><%=rb.getString("FuWuZhiLiangGaoJing")%></span>
							</div>
							<div v-if="scope.row.EVENT_TYPE == '30002'">
								<span><%=rb.getString("ChuLiShiBaiGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30003'">
								<span><%=rb.getString("SheBeiGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30004'">
								<span><%=rb.getString("HuanJingGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30006'">
								<span><%=rb.getString("XingNengYiChuGaoJing")%></span>
							</div>
						</template>
					</el-table-column>
				<el-table-column prop="EXPLANATION" label="<%=rb.getString("GaoJingJieShi")%>" min-width="700" show-overflow-tooltip></el-table-column>
               
            </el-ctable>
          </el-tab-pane>
        </el-tabs>
    </div>
</div>
<script type="text/javascript">
new Vue({
	el:'#alarmLibInfo',
	data(){
		return {
            params:{
                search_text:'',
            },
            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
            libraryTableUrl:'${ctx}/cell/fault/queryAlarmLevelInfosList.action',

		}
	},
    computed:{},
	methods:{
		// 搜索事件
		queryLibrary(val){
			var vm = this;
			vm.params.search_text = val;
		},
		exportAlarmLib(){
			var vm = this;

			exportByForm("${ctx}/cell/fault/exportAlarmLevelResult.action",{
				searchText: vm.params.search_text
			})
		},
		// 严重程度级别重置
		resetAlarmServerity(serverityId,alarmIdentifier){
			var vm = this,
				operatorCode = operatorCodeGloab,
				params={
					operator_code: operatorCode,
					serverityId: serverityId,
					alarmIdentifier:alarmIdentifier
				};
			vm.$confirm('<%=rb.getString("QueDingXiuGaiGaoJingJjiBie")%>','<%=rb.getString("QueRen")%>').then(function(){

						axios.post('${ctx}/cell/fault/updateAlarmServerity.action',stringify(params)).then(function(response){
							var data = response.data;
							if(data) {
								if(data["success"]){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type:'success'
									});
									refreshAliveAlarmCount();
									vm.$refs.libraryTable.refresh()
									
								}else{
									vm.$message.error(data["message"])
								}
							}
						}).catch(function(error){})
					});
		},
	},
	mounted(){}
	
})

</script> 
