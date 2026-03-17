<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>

/*.queryInfo{
	display:inline-block;
	vertical-align:top;
}
.queryInfo label{
	display:block;
	margin-bottom:5px;
	line-height:26px;
}*/
#omcLogs .el-card__body{
	padding:0px;
}
#omcLogs .el-date-editor .el-range__close-icon{
	line-height: normal;
}
.el-tooltip__popper.is-dark { margin: 0 30px; word-break: break-all; word-wrap: break-word; }

</style>
<div class="overflow-cls">
<!-- 网管日志功能页面 -->
<div class="panelDefault" id='omcLogs' style="min-width: 900px;">
	<!-- 按钮  -- 查看旧版本 -->
	<div v-if="showOldLog" class="circleIcon" style="right:60px;">
		<span class="el-icon el-icon-circle-oldData" @click="viewOldLogs"></span>
		<div class="titleButtonText"><%=rb.getString("ChaKan")%></div>
	</div>
	
	<!-- 按钮  -- 导出 -->
	<div class="circleIcon">
		<span class="el-icon el-icon-circle-export" @click="exportLogs"></span>
		<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
	</div>

	<!-- 按钮 -- 关闭旧版本，返回上一层 -->
	<div class='circleIcon' :style='styleObj'>
		<el-button @mouseover.native='showText' @mouseout.native='hideText' @click="cancelSlide" type="primary" :icon='"el-icon-close"' circle></el-button>
		<p class='circleButtonText' v-show='addText'><%=rb.getString("GuanBi")%></p>
	</div>
	
	<!-- 主页面区域 -- tab页 -->
	<el-tabs v-model="activeName" @tab-click='tabClick' style="height: 100%;" v-show="tabsShow">
		<!-- 操作日志 -->
		<el-tab-pane v-if="isSupportOperationLog" label="<%=rb.getString("CaoZuoRiZhi")%>" name="first">
			<el-ctable id="operationLogs" ref="ctableOp" url="${ctx}/system/operatorLog/getOperationLogPageList.action" :height="height" 
					   :query-params="params_operation" pagination="true" :rownumber="true">
				
				<!-- 查询 -- 操作日志 -->
				<template slot="toolbar">
					<el-form :model='params_operation' ref="params_operation" label-position="top" inline=true>
						<el-query  @query="queryOp" @advance-query="advanceQueryOp" @reset="resetQueryOp" :ok-text="queryButton" :reset-text="resetButton"
									placeholder='<%=rb.getString("YongHuMingCheng")%> / <%=rb.getString("IPDiZhi")%> / <%=rb.getString("RiZhiMingCheng")%>' >
							<template slot="form">
								<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="userCode">
									<el-select v-model="params_operation.userCode">
										<el-option v-for="item in userNameOptionsOp" :key="item.id" :label="item.text" :value="item.id">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop="operateIp">
									<el-input v-model="params_operation.operateIp"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("RiZhiMingCheng")%>' prop="logName">
									<el-select v-model="params_operation.logName">
										<el-option v-for="item in operationLogNameOptions" :key="item.id" :label="item.text" :value="item.id">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("XiangXiJiLu")%>' prop="detail">
									<el-input v-model="params_operation.detail"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("JieGuo")%>' prop="result">
									<el-select v-model="params_operation.result" placeholder='<%=rb.getString("SuoYou")%>'>
										<el-option label='<%=rb.getString("SuoYou")%>' value=""></el-option>
										<el-option label='<%=rb.getString("ChengGong")%>' value="1"></el-option>
										<el-option label='<%=rb.getString("ShiBai")%>' value="0"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiBaiYuanYin")%>' prop="reason">
									<el-input v-model="params_operation.reason"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiJian")%>' prop="deteValueOp">
									<el-date-picker type="datetimerange" v-model="dateValueOp" value-format="yyyy-MM-dd HH:mm:ss" 
												start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
								</el-form-item>
							</template>
						</el-query>
					</el-form>
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='<%=rb.getString("YongHuMingCheng")%>' width="150" prop="user_code"></el-table-column>
				<el-table-column label='<%=rb.getString("IPDiZhi")%>' width="200" prop="operate_ip"></el-table-column>
				<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' width="200"  prop="log_name" ></el-table-column>
				<el-table-column label='<%=rb.getString("XiangXiJiLu")%>' prop="detail" min-width="300" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="result" :formatter='resultFmt'></el-table-column>
				<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200" prop="reason" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("CaoZuoKaiShiShiJian")%>' width="150" prop="op_start_time" ></el-table-column>
				<el-table-column label='<%=rb.getString("CaoZuoJieShuShiJian")%>' width="150" prop="op_end_time"></el-table-column>
			</el-ctable>
		</el-tab-pane>

		<!-- 安全日志 -->
		<el-tab-pane v-if="isCloudCore == 'false' && isSupportSecurityLog" label="<%=rb.getString("AnQuanRiZhi")%>" name="second">
			<el-ctable id="securityLogs" ref="ctableSec" url="${ctx}/system/securityLog/getSecurityLogPageList.action" :height="height" 
					   :query-params="params_security" pagination="true" :rownumber="true">
				
				<!-- 查询 -- 安全日志 -->
				<template slot="toolbar">
					<el-form :model='params_security' ref="params_security" label-position="top" inline=true>
						<el-query  @query="querySec" @advance-query="advanceQuerySec" @reset="resetQuerySec" :ok-text="queryButton" :reset-text="resetButton"
								placeholder='ID / <%=rb.getString("YongHuMingCheng")%> / <%=rb.getString("IPDiZhi")%> / <%=rb.getString("RiZhiMingCheng")%>'>
							<template slot="form">
								<el-form-item label='ID' prop="id">
									<el-input v-model="params_security.id"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="userCode">
									<el-select v-model="params_security.userCode">
										<el-option v-for="item in userNameOptionsSec" :key="item.id" :label="item.text" :value="item.id">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop="operateIp">
									<el-input v-model="params_security.operateIp"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("RiZhiMingCheng")%>' prop="logName">
									<el-select v-model="params_security.logName">
										<el-option v-for="item in securityLogNameOptions" :key="item.id" :label="item.text" :value="item.id">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("XiangXiJiLu")%>' prop="detail">
									<el-input v-model="params_security.detail"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("JieGuo")%>' prop="result">
									<el-select v-model="params_security.result" placeholder='<%=rb.getString("SuoYou")%>'>
										<el-option label='<%=rb.getString("SuoYou")%>' value=""></el-option>
										<el-option label='<%=rb.getString("ChengGong")%>' value="1"></el-option>
										<el-option label='<%=rb.getString("ShiBai")%>' value="0"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiBaiYuanYin")%>' prop="reason">
									<el-input v-model="params_security.reason"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiJian")%>'>
									<el-date-picker type="datetimerange" v-model="dateValueSec" value-format="yyyy-MM-dd HH:mm:ss" 
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
								</el-form-item>
							</template>
						</el-query>
					</el-form>
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='ID' width="80" prop="id"></el-table-column>
				<el-table-column label='<%=rb.getString("YongHuMingCheng")%>' width="150" prop="user_code"></el-table-column>
				<el-table-column label='<%=rb.getString("IPDiZhi")%>' width="200" prop="operate_ip"></el-table-column>
				<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' width="200"  prop="log_name" ></el-table-column>
				<el-table-column label='<%=rb.getString("XiangXiJiLu")%>' prop="detail" min-width="300" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="result" :formatter='resultFmt'></el-table-column>
				<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200" prop="reason" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJian")%>' width="150" prop="time"></el-table-column>
			</el-ctable>
		</el-tab-pane>
		
		<!-- 系统日志 -->
		<el-tab-pane v-if="isAdmin == 'true' && isSupportSystemLog" label="<%=rb.getString("XiTongRiZhi")%>" name="third">
			<el-ctable id="systemLogs" ref="ctableSys" url="${ctx}/system/systemLog/getSystemLogPageList.action" :height="height" 
					   :query-params="params_system" pagination="true" :rownumber="true">
				
				<!-- 查询 -- 系统日志 -->
				<template slot="toolbar">
					<el-form :model='params_system' ref="params_system" label-position="top" inline=true>
						<el-query  @query="querySys" @advance-query="advanceQuerySys" @reset="resetQuerySys" :ok-text="queryButton" :reset-text="resetButton"
								placeholder='ID / <%=rb.getString("RiZhiMingCheng")%>'>
							<template slot="form">
								<el-form-item label='ID' prop="id">
									<el-input v-model="params_system.id"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("RiZhiMingCheng")%>' prop="logName">
									<el-select v-model="params_system.logName">
										<el-option v-for="item in systemLogNameOptions" :key="item.id" :label="item.text" :value="item.id">
										</el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("XiangXiJiLu")%>' prop="detail">
									<el-input v-model="params_system.detail"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("JieGuo")%>' prop="result">
									<el-select v-model="params_system.result" placeholder='<%=rb.getString("SuoYou")%>'>
										<el-option label='<%=rb.getString("SuoYou")%>' value=""></el-option>
										<el-option label='<%=rb.getString("ChengGong")%>' value="1"></el-option>
										<el-option label='<%=rb.getString("ShiBai")%>' value="0"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiBaiYuanYin")%>' prop="reason">
									<el-input v-model="params_system.reason"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiJian")%>'>
									<el-date-picker type="datetimerange" v-model="dateValueSys" value-format="yyyy-MM-dd HH:mm:ss" 
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
								</el-form-item>
							</template>
						</el-query>
					</el-form>
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='ID' width="80" prop="id"></el-table-column>
				<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' width="200"  prop="log_name" ></el-table-column>
				<el-table-column label='<%=rb.getString("XiangXiJiLu")%>' prop="detail" min-width="300" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="result" :formatter='resultFmt'></el-table-column>
				<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200" prop="reason" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("ShiJian")%>' width="150" prop="time"></el-table-column>
			</el-ctable>
		</el-tab-pane>
		
		<!-- 北向接口日志 --><!-- url="${ctx}/v1/log/page.action"  -->
		<el-tab-pane v-if="isSupportNorthLog" label="<%=rb.getString("BeiXiangJieKouRiZhi")%>" name="forth">
			<el-ctable id="northLogs" ref="ctableNorth"  :height="height" url="${ctx}/northboundApi/v1/log/page"
					   :query-params="params_north" pagination="true" :rownumber="true">
				
				<!-- 查询 -- 北向接口日志-->  
				<template slot="toolbar">
					<el-form :model='params_north' ref="params_north" label-position="top" inline=true>
						<el-query  
							@query="queryNorth" 
							@advance-query="advanceQueryNorth" 
							@reset="resetQueryNorth" 
							:ok-text="queryButton" 
							:reset-text="resetButton"
							placeholder='<%=rb.getString("IPDiZhi") %>'
						>
							<template slot="form">
								<el-form-item label='<%=rb.getString("IPDiZhi") %>' prop="ipAddress">
									<el-input v-model="params_north.ipAddress"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("RiZhiMingCheng") %>' prop="name">
									<el-input v-model="params_north.name"></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("BeiXiangJieKouLeiXing") %>' prop="type">
									<el-select v-model="params_north.type" placeholder='<%=rb.getString("SuoYou")%>'>
										<el-option label='<%=rb.getString("SuoYou")%>' value=""></el-option>
										<el-option label='<%=rb.getString("TongBuQingQiu") %>' value="1"></el-option>
										<el-option label='<%=rb.getString("YiBuQingQiu") %>' value="2"></el-option>
										<el-option label='<%=rb.getString("GaoJing") %>' value="Real Alarm"></el-option>
										<el-option label='<%=rb.getString("TongBuXiaoXi") %>' value="Sync Msg"></el-option>
										<el-option label='<%=rb.getString("DengLu") %>' value="Login"></el-option>
										<el-option label='<%=rb.getString("TongBuWenJian") %>' value="Sync File"></el-option>
										<el-option label='<%=rb.getString("ShouYe_FeiLianJie") %>' value="Disconnection"></el-option>
										<el-option label='<%=rb.getString("ShouYe_LianJie") %>' value="Connection"></el-option>
										<el-option label='<%=rb.getString("LianJieChaoShi") %>' value="Connection Timeout"></el-option>
										<el-option label='<%=rb.getString("KongXianChaoShi") %>' value="Idle Timeout"></el-option>
										<el-option label='<%=rb.getString("XinTiao") %>' value="HEARTBEAT"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='<%=rb.getString("ShiJian")%>'>
									<el-date-picker type="datetimerange" v-model="dateValueNorth" value-format="yyyy-MM-dd HH:mm:ss" 
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
								</el-form-item>
							</template>
						</el-query>
					</el-form>
					<!--
					<div class="queryGroup">
						<el-input class='pairgrid-query' v-model='searchNorth'  @keyup.enter.native="queryNorth"
								placeholder="<%=rb.getString("IPDiZhi") %>"></el-input>
						<i @click='queryNorth' class="el-icon el-icon-common-search ml10" ></i>
					</div>
					-->
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' width="300"  prop="log_name" >
				
					<template slot-scope="scope">
	            		<span v-if="isLocalZH == true">{{scope.row.nameCn}}</span>
	            		<span v-else>{{scope.row.nameEn}}</span>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("IPDiZhi") %>' width="200"  prop="ipAddress" ></el-table-column>
				<el-table-column label='<%=rb.getString("BeiXiangJieKouLeiXing") %>' width="150"  prop="type" >
					<template slot-scope="scope">
	            		<span v-if="scope.row.type == '1'"><%=rb.getString("TongBuQingQiu") %></span>
	            		<span v-if="scope.row.type == '2'"><%=rb.getString("YiBuQingQiu") %></span>
	            		<span v-if="scope.row.type == 'Real Alarm'"><%=rb.getString("GaoJing") %></span>
	            		<span v-if="scope.row.type == 'Sync Msg'"><%=rb.getString("TongBuXiaoXi") %></span>
	            		<span v-if="scope.row.type == 'Login'"><%=rb.getString("DengLu") %></span>
	            		<span v-if="scope.row.type == 'Sync File'"><%=rb.getString("TongBuWenJian") %></span>
	            		<span v-if="scope.row.type == 'Disconnection'"><%=rb.getString("ShouYe_FeiLianJie") %></span>
	            		<span v-if="scope.row.type == 'Connection'"><%=rb.getString("ShouYe_LianJie") %></span>
	            		<span v-if="scope.row.type == 'Connection Timeout'"><%=rb.getString("LianJieChaoShi") %></span>
	            		<span v-if="scope.row.type == 'Idle Timeout'"><%=rb.getString("KongXianChaoShi") %></span>
	            		<span v-if="scope.row.type == 'HEARTBEAT'"><%=rb.getString("XinTiao") %></span>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("BeiXiangJieKouQingQiu") %>' width="250" prop="reqParams"></el-table-column>
				<el-table-column label='<%=rb.getString("BeiXiangJieKouFanHui") %>'  prop="status" >
					<template slot-scope="scope">
	            		<span v-if="scope.row.type == '2' && scope.row.status == '1' ">{{scope.row.resParams2}}</span>
	            		<span v-else>{{scope.row.resParams1}}</span>
	          		</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("BeiXiangJieKouXiangYingShiJian") %>' width="200" prop="createTime"></el-table-column>
			</el-ctable>
		</el-tab-pane>
	</el-tabs>
	
	<!-- 二级页面 -- 查看旧版本日志 -->
	<el-slide ref="slide" url="${ctx}/system/operationNote/goToOMCLogMain.action" :title="'<%=rb.getString("QuXiao")%>'" 
			  :footer="false" :header='false' position="top" :height="height" :modal='modal'  :width="width" >
	 </el-slide>
	 
	 <form id = "exportLog" style="display:none" method="post"></form>
</div>
</div>
<script type="text/javascript">

var omcLogs = new Vue({
	el:'#omcLogs',
	data:{
		activeName:'',
		showOldLog:true,
		isCloudCore:isCloudCore,
		isAdmin:is_super_user,
		params_operation:{
			timeZone:timeZone,
			logName:'',
			userCode:user_code =='admin'?'':user_code,
			operateIp:'',
			detail:'',
			result:'',
			reason:'',
			startTime:'',
			endTime:'',
			searchText:''
		},
		params_security:{
			timeZone:timeZone,
			id:'',
			logName:'',
			userCode:user_code =='admin'?'':user_code,
			operateIp:'',
			detail:'',
			result:'',
			reason:'',
			startTime:'',
			endTime:'',
			searchText:''
		},
		params_system:{
			timeZone:timeZone,
			id:'',
			logName:'',
			detail:'',
			result:'',
			reason:'',
			startTime:'',
			endTime:'',
			searchText:''
		},
		params_north:{
			timeZone:timeZone,
			searchText:'',
			type: '',
			startTime:'',
			endTime:'',
			name: '',
			ipAddress: '',
			operatorCode: operatorCodeGloab,
			language: language
		},
		searchNorth:"",
		params:'',
	    rowData:[],
	    styleObj:{
	    	zIndex:-10
	    },
	    tabsShow: true,
	    height:'100%',
	    width:'100%',
	    showTip:true,
	    modal:false,
	    operateType:'',
	    addText:false,
	    exportFlag:false,
	    queryButton:'<%=rb.getString("ChaXun")%>',
	    resetButton:'<%=rb.getString("ChaXunChongZhi")%>',
	    dateValueOp:[],
	    dateValueSec:[],
	    dateValueSys:[],
		dateValueNorth: [],
	    userNameOptionsOp:[],
	    userNameOptionsSec:[],
	    operationLogNameOptions:[],
	    securityLogNameOptions:[],
	    systemLogNameOptions:[],
	    ifAddFlag:true,
		taskName:'<%=rb.getString("RenWuMingCheng")%>',
	},
    computed: {
        isSupportOperationLog() {
            return writableMap["CODE_SYSTEM_LOGS_OPERATION"] != undefined;
        },
        isSupportSecurityLog() {
            return writableMap["CODE_SYSTEM_LOGS_SECURITY"] != undefined;
        },
        isSupportSystemLog() {
            return writableMap["CODE_SYSTEM_LOGS_SYSTEM"] != undefined;
        },
        isSupportNorthLog() {
            return writableMap["CODE_SYSTEM_LOGS_NORTH"] != undefined;
        }
    },
	mounted(){
		this.init();
		eventBus.$on('cancel-upgrade',this.closeUpgradeTask);
		eventBus.$on('hide-upgrade',this.hideUpgradeTask);
	},
	methods:{
		// 初始化
		init(){    //根据权限判断页面默认显示的tab内容 
			if(writableMap["CODE_SYSTEM_LOGS_OPERATION"] != undefined){
				this.activeName = 'first'
			}else if(writableMap["CODE_SYSTEM_LOGS_SECURITY"] != undefined){
				this.activeName = 'second'
			}else if(writableMap["CODE_SYSTEM_LOGS_SYSTEM"] != undefined){
				this.activeName = 'third'
			}
		
			var vm = this;
            //获取旧版本日志数据，当数据为空时，不显示入口按钮
			var showOldFlag = 0;
            if(writableMap['CODE_SYSTEM_LOGS_OPERATION'] != undefined){
            	//获取用户名信息 -- 填充下拉选项  -- 操作日志 
                axios.post('${ctx}/system/operatorLog/getUserListForCombobox.action',stringify({
                    
                })).then(function(response){
                    let data = response.data
                    vm.userNameOptionsOp = data;
                }).catch(function(error){})
                //获取日志名称信息 -- 填充下拉选项  -- 操作日志 
                axios.post('${ctx}/system/operatorLog/getOperationNameForCombobox.action',stringify({
                    
                })).then(function(response){
                    let data = response.data
                    vm.operationLogNameOptions = data;
                }).catch(function(error){})
                // 获取旧版本日志数据数量 -- 操作日志
                axios.post('${ctx}/system/operationNote/getOperationNotePageList.action',stringify({
                    timeZone: timeZone
                })).then(function(response){
                    showOldFlag = parseInt(response.data.total);
                }).catch(function(error){})
            }
            if(writableMap['CODE_SYSTEM_LOGS_SECURITY'] != undefined){ 
                //获取用户名信息 -- 填充下拉选项  -- 安全日志 
                axios.post('${ctx}/system/securityLog/getUserListForCombobox.action',stringify({
                    
                })).then(function(response){
                    let data = response.data
                    vm.userNameOptionsSec = data;
                }).catch(function(error){})
                //获取日志名称信息 -- 填充下拉选项  -- 安全日志 
                axios.post('${ctx}/system/securityLog/getOperationNameForCombobox.action',stringify({
                    
                })).then(function(response){
                    let data = response.data
                    vm.securityLogNameOptions = data;
                }).catch(function(error){})
                // 获取旧版本日志数据数量 -- 安全日志
                axios.post('${ctx}/system/logmange/security/getSecurityLogPageList.action',stringify({
                    timeZone:timeZone
                })).then(function(response){
                    showOldFlag = showOldFlag + parseInt(response.data.total);
                }).catch(function(error){})
            } 
            if(writableMap['CODE_SYSTEM_LOGS_SYSTEM'] != undefined){ 
                //获取日志名称信息 -- 填充下拉选项  -- 系统日志  
                axios.post('${ctx}/system/systemLog/getOperationNameForCombobox.action',stringify({
                    
                })).then(function(response){
                    let data = response.data
                    vm.systemLogNameOptions = data;
                }).catch(function(error){})
                // 获取旧版本日志数据数量 -- 系统日志
                axios.post('${ctx}/system/logmange/system/getSystemLogPageList.action',stringify({
                    timeZone:timeZone
                })).then(function(response){
                    showOldFlag = showOldFlag + parseInt(response.data.total);
                }).catch(function(error){})
            }
			vm.showOldLog = showOldFlag == 0 ? false : true ;
		},
		/**
		* 操作“结果”格式化
		* @param row{object}   行数据
		* @param column{object}   列数据
		* @param cellValue{string}   prop绑定值
		* @param index{number}   下标
		*/ 
		resultFmt(row,column,cellValue,index){
			//0 --  失败  1 -- 成功
	    	var status = {
					'0':'<%=rb.getString("ShiBai")%>',
	    			'1':'<%=rb.getString("ChengGong")%>',
	    	}
	    	return status[cellValue]
	    },
		//导出日志数据 
		exportLogs(){     
			var vm = this;
			var activeName = this.$root.activeName
			if(activeName == 'first'){	    		
	    		/* $("#exportLog").form('submit', {
	       	        url: '${ctx}/system/operatorLog/exportLogToCsvFile.action',
	       	        onSubmit: function(params){
	       	        	params.timeZone = timeZone;
	       				params.id = vm.params_operation.id ;
	       				params.userCode = vm.params_operation.userCode;
	       				params.operateIp = vm.params_operation.operateIp;
	       				params.logName = vm.params_operation.logName;
	       				params.detail = vm.params_operation.detail;
	       				params.result = vm.params_operation.result;
	       				params.reason = vm.params_operation.reason;
	       				params.startTime = vm.params_operation.startTime;
	       				params.endTime = vm.params_operation.endTime;
	       				params.searchText = vm.params_operation.searchText;
	       	        }
	       	    });  */ 
				var params = {};
	       	 	params.timeZone = timeZone;
   				params.userCode = vm.params_operation.userCode;
   				params.operateIp = vm.params_operation.operateIp;
   				params.logName = vm.params_operation.logName;
   				params.detail = vm.params_operation.detail;
   				params.result = vm.params_operation.result;
   				params.reason = vm.params_operation.reason;
   				params.startTime = vm.params_operation.startTime;
   				params.endTime = vm.params_operation.endTime;
   				params.searchText = vm.params_operation.searchText;
				exportByForm('${ctx}/system/operatorLog/exportLogToCsvFile.action',params);
	    	}else if(activeName == 'second'){
	    		/* $("#exportLog").form('submit', {
	       	        url: '${ctx}/system/securityLog/exportLogToCsvFile.action',
	       	        onSubmit: function(params){
	       	        	params.timeZone = timeZone;
	       				params.id = vm.params_security.id ;
	       				params.userCode = vm.params_security.userCode;
	       				params.operateIp = vm.params_security.operateIp;
	       				params.logName = vm.params_security.logName;
	       				params.detail = vm.params_security.detail;
	       				params.result = vm.params_security.result;
	       				params.reason = vm.params_security.reason;
	       				params.startTime = vm.params_security.startTime;
	       				params.endTime = vm.params_security.endTime;
	       				params.searchText = vm.params_security.searchText;
	       	        }
	       	    });  */
				var params = {};
	       	 	params.timeZone = timeZone;
   				params.id = vm.params_security.id ;
   				params.userCode = vm.params_security.userCode;
   				params.operateIp = vm.params_security.operateIp;
   				params.logName = vm.params_security.logName;
   				params.detail = vm.params_security.detail;
   				params.result = vm.params_security.result;
   				params.reason = vm.params_security.reason;
   				params.startTime = vm.params_security.startTime;
   				params.endTime = vm.params_security.endTime;
   				params.searchText = vm.params_security.searchText;
				exportByForm('${ctx}/system/securityLog/exportLogToCsvFile.action',params);
	    	}else if(activeName == 'third'){
	    		/* $("#exportLog").form('submit', {
	       	        url: '${ctx}/system/systemLog/exportLogToCsvFile.action',
	       	        onSubmit: function(params){
	       	        	params.timeZone = timeZone;
	       				params.id = vm.params_system.id ;
	       				params.logName = vm.params_system.logName;
	       				params.detail = vm.params_system.detail;
	       				params.result = vm.params_system.result;
	       				params.reason = vm.params_system.reason;
	       				params.startTime = vm.params_system.startTime;
	       				params.endTime = vm.params_system.endTime;
	       				params.searchText = vm.params_system.searchText;
	       	        }
	       	    });  */
				var params = {};
				params.timeZone = timeZone;
   				params.id = vm.params_system.id ;
   				params.logName = vm.params_system.logName;
   				params.detail = vm.params_system.detail;
   				params.result = vm.params_system.result;
   				params.reason = vm.params_system.reason;
   				params.startTime = vm.params_system.startTime;
   				params.endTime = vm.params_system.endTime;
   				params.searchText = vm.params_system.searchText;
				exportByForm('${ctx}/system/systemLog/exportLogToCsvFile.action',params);
	    	}else if(activeName == 'forth') {
				var params = {};
				params.timeZone = timeZone;
				params.startTime = vm.params_north.startTime;
   				params.endTime = vm.params_north.endTime;
   				params.type = vm.params_north.type;
   				params.searchText = vm.params_north.searchText;
				params.name = vm.params_north.name;
				params.ipAddress = vm.params_north.ipAddress;
				params.operatorCode = vm.params_north.operatorCode;
				params.language = language;
				exportByForm('${ctx}/northboundApi/v1/log/exportLogToCsvFile',params);
			}
		},
		queryOp:function(val){   //模糊查询
			this.resetQueryOp();
			this.params_operation.searchText  = val;
			this.$refs.ctableOp.refresh()
		},
		querySec:function(val){   //模糊查询
			this.resetQuerySec();
			this.params_security.searchText  = val;
			this.$refs.ctableSec.refresh()
		},
		querySys:function(val){   //模糊查询
			this.resetQuerySys();
			this.params_system.searchText  = val;
			this.$refs.ctableSys.refresh()
		},
		queryNorth:function(val){
			this.params_north.searchText = val;
			this.searchNorth = val;
			this.$refs.ctableNorth.refresh();	 
		},
		advanceQueryOp:function(){  //高级查询  -- 操作日志 
			this.params_operation.searchText = "";
			if(this.dateValueOp != null){
				this.params_operation.startTime = this.dateValueOp[0];
    			this.params_operation.endTime = this.dateValueOp[1];
			}else{
				this.params_operation.startTime = '';
    			this.params_operation.endTime = '';
			}
			this.$refs.ctableOp.refresh()
		},
		advanceQuerySec:function(){  //高级查询  -- 安全日志 
			this.params_security.searchText = "";
			if(this.dateValueSec != null){
				this.params_security.startTime = this.dateValueSec[0];
    			this.params_security.endTime = this.dateValueSec[1];
			}else{
				this.params_security.startTime = '';
    			this.params_security.endTime = '';
			}
			this.$refs.ctableSec.refresh()
		},
		advanceQuerySys:function(){  //高级查询  -- 系统日志 
			this.params_system.searchText = "";
			if(this.dateValueSys != null){
				this.params_system.startTime = this.dateValueSys[0];
    			this.params_system.endTime = this.dateValueSys[1];
			}else{
				this.params_system.startTime = '';
    			this.params_system.endTime = '';
			}
			this.$refs.ctableSys.refresh()
		},
		advanceQueryNorth() {
			this.params_north.searchText = "";
			if(this.dateValueNorth != null){
				this.params_north.startTime = this.dateValueNorth[0];
    			this.params_north.endTime = this.dateValueNorth[1];
			}else{
				this.params_north.startTime = '';
    			this.params_north.endTime = '';
			}
			this.$refs.ctableNorth.refresh()
		},
		resetQueryOp:function(){   //查询重置  -- 操作日志 
			this.$refs.params_operation.resetFields();
		    this.dateValueOp = [];
		},
		resetQuerySec:function(){  //查询重置  -- 安全日志 
			this.$refs.params_security.resetFields();
			this.dateValueSec = [];
		},
		resetQuerySys:function(){  //查询重置  -- 系统日志 
			this.$refs.params_system.resetFields();
			this.dateValueSys = [];
		},
		resetQueryNorth() {
			this.$refs.params_north.resetFields();
			this.dateValueNorth = [];
		},
		viewOldLogs(){    //查看旧版本日志 
			var vm = this;
		
			vm.$root.styleObj = {
	    			zIndex:2002,
	    	}

    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
	    	
		},
	    cancelSlide(){    //退出旧版本日志，返回网管日志主页面 
			var vm = this;
			vm.$root.styleObj = {
	    			zIndex:-10
	    	}
			eventBus.$emit('hander-close')
    		this.$refs.slide.hide();
	    	
	    },
		// 鼠标移入
	    showText(){
	    	this.addText = true;
	    },
		// 鼠标移出
	    hideText(){
	    	this.addText = false;
	    },
		// 导航切换
	    tabClick(tab){
	    	this.$refs.slide.hide();
	    	eventBus.$emit('hander-close');
	    }
	}
})
</script>