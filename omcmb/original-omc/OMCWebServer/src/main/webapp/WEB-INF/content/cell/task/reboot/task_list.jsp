<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 重启任务列表界面 --%>
<style type="text/css">

#rebootTaskContent .el-input-small{
	width:400px;
}
#rebootTaskContent .queryInfo label{
	display:block;
	margin-bottom:5px;
	line-height:26px;
}
#rebootTaskContent .queryInfo{
	display:inline-block;
}
#rebootTaskContent .el-form-item__error{
	margin-top:5px;
}

#rebootTaskContent .el-date-editor .el-range-separator{
	line-height:24px;
}

#rebootTaskContent .el-date-editor .el-range__icon{
	line-height:24px;
}

.enb-menu-list {
	color: #000;
	width:160px;
	border:1px solid rgba(188,188,188,0.1);
	background:#fff;
	position:absolute;
	top:52px;
	right:25px;
	box-shadow:0 5px 15px #d8d8d8;
}
.enb-menu-list>div{
	position: relative;
	height:40px;
	line-height:40px;
	padding-left:30px;
	border-bottom:1px solid #e5f0f6; 
}
.enb-menu-list div a{
	color:#000;
}
.enb-menu-list div:last-of-type{
	border:none;
}
.enb-menu-list>div:hover{
	background:#e1f2fa;
}
.enb-menu-list div:active{
	background:#c4e6f5;
}
.el-pagination .el-select .el-input{
	width:88px;
}
.el-pagination .el-select .el-input .el-input__inner{
	width:88px;
	background:#fff !important;
}

.enbH100{
	height:100%
}
.enbW280{
	width:280px
}
.ml50{
	margin-left:50px
}
.enbExportBtn{
	width:25px;
	height:20px;
	position:absolute;
	right:45px;
	top:0px
}
.enbCurpo{
	cursor: pointer
}
#rebootTaskContent .taskAndDeviceTabs{
	position: absolute;
	top: 18px;
	left: 30px;
	display: flex;
	z-index: 30;
}
#rebootTaskContent .taskAndDeviceTabs div{
	box-sizing: border-box;
	cursor: pointer;
}
#rebootTaskContent .tabTaskDefaultCls{
	height: 30px;
	padding: 0px 20px;
	display: flex;
	align-items: center;
	border: 1px solid #E9E9E9;
	border-right: none;
	border-radius: 4px 0px 0px 4px;
	color: #666666;
}
#rebootTaskContent .tabTaskDefaultCls .el-icon::before, #rebootTaskContent .tabDeviceDefaultCls .el-icon::before{
	color: #666666;
}
#rebootTaskContent .tabTaskSelectCls{
	height: 30px;
	padding: 0px 20px;
	display: flex;
	align-items: center; 
	border-left: none;
	border-radius: 4px 0px 0px 4px;
	color: #FFFFFF;
}
#rebootTaskContent .tabTaskSelectCls .el-icon::before,#rebootTaskContent .tabDeviceSelectCls .el-icon::before{
	color: #FFFFFF;
}
#rebootTaskContent .tabDeviceDefaultCls{
	height: 30px;
	padding: 0px 20px;
	display: flex;
	align-items: center;
	border: 1px solid #E9E9E9;
	border-left: none;
	border-radius: 0px 4px 4px 0px;
	color: #666666;
}
#rebootTaskContent .tabDeviceSelectCls{
	height: 30px;
	padding: 0px 20px;
	display: flex;
	align-items: center;
	border-left: none;
	border-radius: 0px 4px 4px 0px;
	color: #FFFFFF;
}
#rebootTaskContent .flex-form {
	display: flex;
	flex-wrap: wrap;
}
#rebootTaskContent .flex-form .el-form-item {
	margin-right: 100px;
	margin-bottom: 10px;
}
#rebootTaskContent .taskListTable{
	position: absolute;
	top: 5px;
	bottom: 0px;
	left: 0px;
	right: 0px;
	z-index: 20;
}
#rebootTaskContent .deviceListTable{
	position: absolute;
	top: 5px;
	bottom: 0px;
	left: 0px;
	right: 0px;
	z-index: 20;
}
#rebootTaskContent .recurringRebootSwitchBox{
	height: 30px;
	display: flex;
	align-items: center;
	position: absolute;
	top: 12px;
	right:60px;
	z-index: 20;
}
#rebootTaskContent .exportResultDiv{
	position: absolute;
	top: 2px;
	right: 0px;
	z-index: 20;
}
#rebootTaskContent .resultHeadBox{
	position: absolute;
	top:12px;
	right: 100px;
	display: flex;
	z-index: 99;
}
#rebootTaskContent .resultHeadBox .statisticsSuccessDiv{
	height: 30px;
	display: flex;
	align-items: center;
	overflow: hidden;
}
#rebootTaskContent .resultHeadBox .statisticsFailDiv{
	height: 30px;
	display: flex;
	align-items: center;
	padding: 0;
	overflow: hidden;
}
#rebootTaskContent .statisticsSuccessDiv .el-icon::before{
	font-size: 16px;
	color:#67D972;
}
#rebootTaskContent .statisticsSuccessDiv > div:first-child{
	height: 30px;
	display: flex;
	align-items: center;
	border-radius: 4px;
	font-size: 12px;
}
#rebootTaskContent .statisticsSuccessDiv > div:last-child{
	padding: 0px 10px;
	border-right: 1px solid #dfe2ee;
}
#rebootTaskContent .statisticsFailDiv .el-icon::before{
	font-size: 16px;
	color: #E88282;
}
#rebootTaskContent .statisticsFailDiv > div:first-child{
	height: 30px;
	display: flex;
	align-items: center;
	border-radius: 4px;
	font-size: 12px;
	box-sizing: border-box;
}
#rebootTaskContent .statisticsFailDiv > div:last-child{
	padding: 0px 10px;
	height: 30px;
	line-height: 30px;
}
#rebootTaskContent .el-icon-operation-export:before{
	font-size: 16px;
	color: var(--main-color);
}
#rebootTaskContent .profileAddDiv .el-icon-status-timeOut:before { color: #19D5F3; }
</style>

<!-- eNb重启任务 -->
<div class="panelDefault" id='rebootTaskContent'>
	<el-tabs v-model='activeName' class="enbH100" @tab-click='tabClick'>
		<el-tab-pane  name='reboot' label="<%=rb.getString("ChongQi")%>">
			<div class="circleIcon placeholder-bt CODE_ENB_REBOOT hidden" style="z-index:10;top:12px;right:20px;" :placeholder="buttonText" @click="addRebootTask" v-clickoutside="hideChoseList">		
				<span class="el-icon" :class='buttonIcon'></span>
			</div>
			<!-- 任务列表 -->
			<table-temp :id="'enb_reboot_list'"  :params='params_reboot' ref='reboot'></table-temp>
		</el-tab-pane>
		<el-tab-pane name='periodicReboot' label="<%=rb.getString("DingShiChongQi")%>"  v-if="recurringRebootShow"> 
			<div class="periodicRebootTableBox">
				<div class="taskAndDeviceTabs">
					<div :class="taskAndDeviceShow == 'task' ? 'tabTaskSelectCls' : 'tabTaskDefaultCls'" @click="taskAndDeviceTabChange('task')" >
						<span class="el-icon el-icon-taskList" style="margin-right:10px;"></span><%=rb.getString("RenWuLieBiao")%>
					</div>
					<div :class="taskAndDeviceShow == 'device' ? 'tabDeviceSelectCls' : 'tabDeviceDefaultCls'" @click="taskAndDeviceTabChange('device')" >
						<span class="el-icon el-icon-devicelist" style="margin-right:10px;"></span><%=rb.getString("BackupRestoreSheBeiLieBiao")%>
					</div>
				</div>
				<div class="taskListTable" v-show="taskAndDeviceShow == 'task'">
					<div class="recurringRebootSwitchBox">
						<el-switch v-model="recurringRebootSwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#CFCFCF" @change="recurringRebootSwitchChange"></el-switch>
						<span style="margin:0px 10px;width:40px" v-show="recurringRebootSwitch == 0"><%=rb.getString("JinYong")%></span>
						<span style="margin:0px 10px;width:40px" v-show="recurringRebootSwitch == 1"><%=rb.getString("QiYong")%></span>
						<%-- <span style="color:#666666"><%=rb.getString("DingShiRenWuTiShi")%></span> --%>
					</div>
					<div class="circleIcon placeholder-bt profileAddDiv" style="z-index:10;top:12px;right:20px;" placeholder="<%=rb.getString("SheZhi")%>" @click="settingRecurringReboot">		
						<span class="el-icon el-icon-operation-reboot" style='font-size: 14px;'></span>
						<span class='el-icon el-icon-status-timeOut' style='position: absolute; top: 8px; right: 0; font-size: 12px; '></span>
					</div>

					<el-ctable ref="recurringRebootTaskTable" id="recurringRebootTaskTable" time=6 :url="taskListTableUrl" :height="height" :page-size="pageSize" :page-list="pageList" :query-params="queryTaskParams" pagination="true">
						<template slot="toolbar">
							<el-query  @query="queryTask" 
								@advance-query="taskAdvanceQuery" 
								@reset="resetTaskQuery" placeholder="<%=rb.getString("RenWuMingCheng")%>" 
								ok-text="<%=rb.getString("ChaXun")%>" 
								:reset-text="'<%=rb.getString("ChaXunChongZhi")%>'"
								style="padding-left:280px;">
								<template slot="form">
									<el-form :model="queryTaskForm" ref="queryTaskForm" label-position="top" class="flex-form">
										<el-form-item label="<%=rb.getString("RenWuMingCheng")%>" prop="taskName" >
											<el-input v-model="queryTaskForm.taskName" size="mini"></el-input>
										</el-form-item>
										<el-form-item label="<%=rb.getString("KaiShiShiJian")%>" prop="timeRange" >
											<el-date-picker  v-model="timeRange" type="datetimerange" size="mini" value-format="yyyy-MM-dd HH:mm:ss"
												start-placeholder="<%=rb.getString("KaiShiShiJian")%>" end-placeholder="<%=rb.getString("JieShuShiJian")%>"></el-date-picker> 
										</el-form-item>
									</el-form>
								</template>
							</el-query>
						</template>
						
						<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' min-width="300"  prop="taskName" show-overflow-tooltip="true"></el-table-column>
						<el-table-column label='<%=rb.getString("CaoZuoRen")%>' width="80" prop="creator"></el-table-column>
						<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' width="200" prop="createTime"></el-table-column>
						<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="120" prop="taskStatus">
							<template slot-scope="scope">
								<div v-html="taskTableStatus(scope.row.taskStatus)"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("JinDu")%>' width="100" prop="taskProgress"></el-table-column>
						<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="taskResult" :formatter='taskResultFmt'></el-table-column>
						<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="startTime"></el-table-column>
						<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="endTime" width="180"></el-table-column>
					</el-ctable>
				</div>
				<div class="deviceListTable" v-show="taskAndDeviceShow == 'device'">
					<div class="exportResultDiv">
						<div class="placeholder-bt circleIcon" placeholder="<%=rb.getString("DaoChu")%>">
							<span class="el-icon el-icon-circle-export" @click="exportDeviceListTable" style='font-size: 14px;'></span>
						</div>
						
					</div>
					<div class="circleIcon placeholder-bt profileAddDiv" style="z-index:10;top:12px;right:55px;" placeholder="<%=rb.getString("SheZhi")%>" @click="settingRecurringReboot">		
						<span class="el-icon el-icon-operation-reboot" style='font-size: 14px;'></span>
						<span class='el-icon el-icon-status-timeOut' style='position: absolute; top: 8px; right: 0; font-size: 12px; '></span>
					</div>

					<div class="resultHeadBox">
						<div class="statisticsSuccessDiv">
							<div><span class="el-icon el-icon-circle-success" style="margin-right:5px;"></span><%=rb.getString("ChengGong")%></div>
							<div>{{statisticsSuccess}}</div>
						</div>
						<div class="statisticsFailDiv fail_count">
							<div><span class="el-icon el-icon-circle-close" style="margin-right:5px;"></span><%=rb.getString("ShiBai")%></div>
							<div>{{statisticsFail}}</div>
						</div>
					</div>
					<el-ctable 
						id="recurringRebootDeviceTable"
						ref="recurringRebootDeviceTable" 
						:url="queryDeviceUrl" 
						:query-params="queryDeviceParams"
						@load-success="deviceTableLoadSuccess" 
						height="100%"
						time=6 :page-size="pageSize" :page-list="pageList"
						:pagination=true>
						
						<template slot="toolbar">
							<el-query type="normal" @query="queryDeviceList" placeholder='<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("RenWuMingCheng")%>' style="margin-left:280px;"></el-query>
						</template>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' width="220"  prop="serialNumber"></el-table-column>
						<el-table-column label='<%=rb.getString("HostName")%>' width="220" prop="cellName"></el-table-column>
						<el-table-column label='<%=rb.getString("RenWuMingCheng") %>' width="220" prop="taskName"></el-table-column>
						<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="150" prop="progressStatus">
							<template slot-scope="scope">
								<div v-html="resultTableStatus(scope.row.progressStatus)"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("JieGuo")%>' width="150" prop="progressResult" :formatter='deviceResultFmt'></el-table-column>
						<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>'  prop="failureReason" width="200"></el-table-column>
						<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' prop="startTime"></el-table-column>
						<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="endTime"></el-table-column>
					</el-ctable>
				</div>
			</div>
		</el-tab-pane>
		<el-tab-pane name='silenceReboot' label="<%=rb.getString("WuGanZhiChongQi")%>" v-if="silenceRebootShow == 'true'">
			<div id="silenceReboot">

			</div>
		</el-tab-pane>
	</el-tabs>
	<!-- 新建、查看、修改任务浮层 -->
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	 :height="slideHeight" :modal='modal'  :width="slideWidth" :subloading="slideSubmitLoading" @ok='saveRebootTask' @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 	<template slot='toolbar' v-if='exportFlag'>
	 		<a href='#' @click='exportMsg' class='el-icon el-icon-operation-export enbExportBtn'></a>
	 	</template>
	 </el-slide>
	 <el-slide ref="recurringRebootSlide" :url="recurringRebootSlideUrl" :title="recurringRebootSlideTitle" :footer="recurringRebootSlideFooter" :header='recurringRebootSlideHeader' :position="recurringRebootSlidePosition"
	 	:height="recurringRebootSlideHeight" :modal='modal'  :width="recurringRebootSlideWidth" :subloading="slideSubmitLoading" @ok='saveRecurringRebootTask' @cancel='cancelRecurringRebootSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 </el-slide>
	 <!--系统 进程 子机 下拉列表 -->
	 <div id="reboot_choseList" class="enb-menu-list" v-show=showList>
	 	<div @click="createRebootTask(item.value)" v-for="item in choseList"><a href="javascript:void(0)">{{item.text}}</a></div>
	 </div>
</div>
<!-- 任务列表 -->
<template id='rebootTableTemplate'>
	<div class="enbH100">
		<el-ctable ref="ctable" :id="tableId" time=6 :url="url" :height="height" :query-params="params" pagination="true">
			<template slot="toolbar">
				<el-query  @query="query" @advance-query="advanceQuery" @reset="resetQuery" :placeholder="taskName" :arrow-text="advanceText" :ok-text="queryButton" :reset-text="resetButton">
					<template slot="form">
						<div class='queryInfo'>
							<label><%=rb.getString("RenWuMingCheng")%></label>
							<el-input v-model="params.taskName" :value="params.taskName" size="mini" class="enbW280"></el-input>
						</div>
						<div class='queryInfo ml50' >
							<label><%=rb.getString("KaiShiShiJian")%></label>
							<el-date-picker v-model="dateValue" value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
						</div>
					</template>
				</el-query>
			</template>
			<el-table-column label='' width="30" class-name="operationColumn">
				<template slot-scope="scope">
	            	<div class="el-icon el-icon-operation-more enbCurpo" @click="optClick(scope.row,event)" v-clickoutside="handerClose" ></div>
	          	</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' min-width="300"  prop="TASK_NAME" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("CaoZuoRen")%>' width="80" prop="CREATE_USER"></el-table-column>
			<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' width="200" prop="CREATE_TIME"></el-table-column>
			<el-table-column label='<%=rb.getString("ChangPinXingHao")%>' width="200" prop="PRODUCT_TYPE"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="120" prop="TASK_STATUS">
				<template slot-scope="scope">
					<div v-html="backupRestoreTaskTableStatus(scope.row.TASK_STATUS)"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("JinDu")%>' width="100" prop="TASK_PROGRESS"></el-table-column>
			<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="TASK_RESULT" :formatter="resultFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="START_TIME"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="END_TIME" width="180"></el-table-column>
		</el-ctable>
		<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
	</div>
</template>
<script type="text/javascript">
var vmReboot = new Vue({
	el:'#rebootTaskContent',
	data:{
		taskAndDeviceShow:'task',
		height:'100%',
		recurringRebootSwitch:'0',
		taskListTableUrl:'${ctx}/task/recurringReboot/getRebootTaskList.action?operatorCode='+operator_code,
		queryTaskParams:{
			searchText:'',
			timeZone:timeZone,
			taskName:'',
			startTime:'',
			endTime:'',
		},
		timeRange:[],
		queryTaskForm:{
			taskName:'',
			startTime:'',
			endTime:'',
		},
		queryDeviceUrl:'${ctx}/task/recurringReboot/getRebootDeviceTaskProgress.action?operatorCode='+operator_code,
		queryDeviceParams:{
			timeZone: timeZone,
			searchText:'',
		},
		params_reboot:{
			timeZone:timeZone,
			taskName:'',
			startTime:'',
			endTime:'',
			searchText:'',
			likeFields:'task_name'
		},
		statisticsSuccess:'0',
		statisticsFail:'0',
	    rowData:[],
	    slideUrl:'',
	    slideTitle:'',
	    slideHeader:'',
	    slideFooter:'',
	    slidePosition:'',
	    slideHeight:'',
	    slideWidth:'',
        slideSubmitLoading:'',

		recurringRebootSlideUrl:'',
		recurringRebootSlideTitle:'',
	    recurringRebootSlideHeader:'',
	    recurringRebootSlideFooter:'',
	    recurringRebootSlidePosition:'',
	  	recurringRebootSlideHeight:'',
	    recurringRebootSlideWidth:'',
	    showTip:true,
	    modal:false,
	    taskStatus:'',
	    operateType:'',
	    addText:false,
	    exportFlag:false,
	    buttonIcon:'el-icon-circle-add',
	    buttonText:'<%=rb.getString("TianJia") %>',
	    ifAddFlag:true,
	    styleObj:{
	    	zIndex:10
	    },
	    showList:false,
	    choseList:[
	    	{"text":"系统",value:"reboot_system"},
			{"text":"进程",value:"reboot_stk"},
			{"text":"子机",value:"reboot_ru"}
			],
		activeName:'reboot',
		pageSize:50,
		pageList:[50,100,200],
		silenceRebootShow:'false',
	},
	components:{
	    'table-temp':{
	    	template:'#rebootTableTemplate',
	    	data(){
	    		return {
    				dateValue:[],
    				menus:[],
    				taskName:'<%=rb.getString("RenWuMingCheng")%>',
    				url:'${ctx}/task/reboot/getRebootTaskList.action',
    				height:'100%',
    				advanceText:'<%=rb.getString("GaoJiChaXun")%>',
    				queryButton:'<%=rb.getString("ChaXun")%>',
    				resetButton:'<%=rb.getString("ChaXunChongZhi")%>',
    				tableId: this.id
	    		}
	    	},
	    	props:['params','id'],
	    	methods:{
				/**
				 * 软件升级模糊查询
				 * @param val:当前搜索数据项
				 
				*/
	    		query:function(val){//
	    			this.resetQuery();
	    			this.params.searchText  = val;
	    			this.$refs.ctable.refresh()
	    		},
	    		advanceQuery:function(){//软件升级高级查询
	    			this.params.searchText = "";
	    			if(this.dateValue != null){
	    				this.params.startTime = this.dateValue[0];
		    			this.params.endTime = this.dateValue[1];
	    			}else{
						this.params.startTime = '';
		    			this.params.endTime = '';
					}
	    			this.$refs.ctable.refresh()
	    		},
	    		resetQuery:function(){//软件升级重置查询
	    			this.params.taskName = '';
	    			this.dateValue = []
	    		},
				/**
				 * 软件升级 任务结果fmt
				 * @param row:数据项  没用
				 * @param column:表头dom 没用
				 * @param cellValue:数据里的cellValue属性进行转化
				 * @param index:下标 没用
				*/
	    	    resultFmt(row,column,cellValue,index){//
	    	    	var resultObj = {
	    					"1" : "<%=rb.getString("ChengGong")%>",
	    					"2" : "<%=rb.getString("BuFenChengGong")%>",
	    					"3" : "<%=rb.getString("ShiBai")%>",
	    					"" : ""
	    			}
	    			return resultObj[cellValue];
	    	    },
	    	    handerClose(){//软件升级点击页面其他地方菜单收起
	    	        this.$refs.menu.hide();
	    	    },
				/**
				 * 点击操作出现下拉菜单
				 * @param row:当前点击数据的 
				 *  1.等待   2.进行中  3.暂停  4.已结束 5.终止中 6.暂停中
				*/
	    		optClick(row,ev){
	    	    	var status = row.TASK_STATUS;
	    	    	this.taskStatus = row.TASK_STATUS;
	    		    this.$root.rowData = row;
    		    	this.menus= [
    			          {label:'<%=rb.getString("JieGuo")%>',code:'view'},
    			          {label:'<%=rb.getString("KaiShi")%>',cls:"CODE_ENB_REBOOT hidden",code:'start'},
    			          {label:'<%=rb.getString("ZanTing")%>',cls:"CODE_ENB_REBOOT hidden" ,code:'stop'},
    			          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"CODE_ENB_REBOOT hidden",code:'terminate'},
    			          {label:'<%=rb.getString("ShanChu")%>',cls:"CODE_ENB_REBOOT hidden",code:'del'}
    			    ]
    		    	var vm = this;
    		    	initTaskStatus(status,vm.menus);
    		    	this.$nextTick(function(){
    		    		document.body.click();
	    		    	vm.$refs.menu.show(ev);
    		    	});
	    	    },
				/**
				 * 点击操作出现下拉菜单
				 * @param ev: 点击当前项数据是哪个
				 * */
	    	    clickMenu(ev){
	    	    	var codes = {
	    	    		view:this.viewRebootTask,
	    	    		start:this.activeRebootTask,
	    	    		stop:this.suspendRebootTask,
	    	    		terminate:this.terminateRebootTask,
	    	    		del:this.delRebootTask
	    	    	}
	    	    	if(codes[ev.code]){
	    	    		codes[ev.code](this.$root.rowData["TASK_ID"],this.$root.rowData["TASK_STATUS"])
	    	    	}
	    	    },
				/**
				 * 执行结果
				 * @param task_id: 当前数据ID
				*/
	    	    viewRebootTask(task_id){ 
	    	    	var vm = this;
	    	    	vm.$root.slideUrl = '${ctx}/task/reboot/toRebootTaskProgress.action?task_id=' + task_id;
	    	    	vm.$root.slideTitle = '<%=rb.getString("ZhiXingJieGuo")%>';
	    	    	vm.$root.slideFooter = false;
	    	    	vm.$root.slideHeader = true;
	    	    	vm.$root.slidePosition = 'bottom';
	    	    	vm.$root.slideHeight = '350px';
	    	    	vm.$root.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
	    	    	vm.$root.operateType='view';
	    	    	vm.$root.exportFlag = true;
	    	    	vm.$root.$refs.slide.showSlide(function(){
	    	    		vm.$root.modal = false;
	    	    	});
	    	    },

				/**
				 * 开始执行任务函数
				 * @param task_id: 当前数据ID
				*/
	    	    activeRebootTask(task_id){ 
	    	    	var vm = this;

	    	    	axios.post('${ctx}/task/reboot/activeTask.action',stringify({
	    	    		taskId : task_id,
	    	    	})).then(function(response){
	    	    		var data = response.data;
	    	    		if(data["success"]){
	    	    			vm.$refs.ctable.refresh()
	    	    		}else{
	    	    			vm.$message.error(data["message"])
	    	    		}
	    	    	})
	    	    },
				/**
				 * 暂停任务函数
				 * @param task_id: 当前数据ID
				*/
	    	    suspendRebootTask(task_id){
	    	    	var vm = this;
	    	    	
	    	    	axios.post('${ctx}/task/reboot/suspendTask.action',stringify({
	    	    		taskId : task_id,
	    	    	})).then(function(response){
	    	    		var data = response.data;
	    	    		if(data["success"]){
	    	    			vm.$refs.ctable.refresh()
	    	    		}else{
	    	    			vm.$message.error(data["message"])
	    	    		}
	    	    	})
	    	    },
				/**
				 * 终止任务函数
				 * @param task_id: 当前数据ID
				*/
	    	    terminateRebootTask(task_id){ 
	    	    	var vm = this;
	    	    	axios.post('${ctx}/task/reboot/terminateRebootTask.action',stringify({
	    	    		taskId : task_id,
	    	    	})).then(function(response){
	    	    		var data = response.data;
	    	    		if(data["success"]){
	    	    			vm.$refs.ctable.refresh()
	    	    		}else{
	    	    			vm.$message.error(data["message"])
	    	    		}
	    	    	})
	    	    },
				/**
				 * 软件升级  删除任务
				 * @param task_id: 当前数据ID
				*/
	    	    delRebootTask(task_id){
	    	    	var vm = this;
	    	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
	    	    		customClass:'warningConfirm',
	    	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    	    		type:'warning',
	    	    		closeOnClickModal:false
	    	    	}).then(() => {
	    	    		axios.post('${ctx}/task/reboot/delRebootTask.action',stringify({
	    		    		taskId:task_id,
	    		    	})).then(function(response){
	    		    		var data = response.data;
	    		    		if(data["success"]){
	    		    			vm.$refs.ctable.refresh()
	    		    			vm.$message({
	    			    			type:'success',
	    			    			message:'<%=rb.getString("ChengGong")%>'
	    			    		})
								vm.$refs.slide.hide();//结果隐藏
	    		    		}else{
	    		    			vm.$message.error(data["message"])
	    		    		}
	    		    	}).catch(function(error){
	    		    		
	    		    	})
	    	    	}).catch()
	    	    }
	    	}
	    }
	},
	computed:{
		recurringRebootShow(){
			return is_super_user == 'true' || is_build_user == 'true'? true : false;
		}
    },
	mounted(){
		this.getSwitchData();
		this.getSilenceRebootStaus();
		eventBus.$off("cancel-reboot").$on('cancel-reboot',this.closeRebootTask);
		eventBus.$off("hide-reboot").$on('hide-reboot',this.hideRebootTask);
		eventBus.$off("hide-recurringReboot-slide").$on('hide-recurringReboot-slide',this.closeRecurringRebootSlide);
	},
	methods:{
		// Tab切换事件
		tabClick(tab){
			var vm = this,
				activeName = this.$root.activeName;

			vm.$refs.slide.hide();//结果隐藏

			if(activeName == 'silenceReboot'){	    		
				$("#silenceReboot").load('${ctx}/task/secretReboot/toRebootTaskList.action',function(data){
					$.parser.parse(this);
				});
	    	}
		},
		// 获取定时重启开关状态
		getSwitchData(){
			var vm =this,
				params={
					operatorCode:operator_code
				};
			axios.post('${ctx}/task/recurringReboot/getSwitch.action',stringify(params)).then(function(response){
				var data = response.data;
				vm.recurringRebootSwitch = data.switch;
			}).catch(function(error){})
		},
		getSilenceRebootStaus(){
			var vm = this;
			
			axios.post('${ctx}/task/recurringReboot/getSecretRebootPermission.action').then(function(response){
				var data = response.data;
				vm.silenceRebootShow = data.permission;
			}).catch(function(error){})
		},
		// 设备表格 加载成功回调
		deviceTableLoadSuccess(data){
			var vm = this;
			vm.statisticsSuccess = data.properties.success;
			vm.statisticsFail = data.properties.fail;
		},
		// 导出定时重启任务结果导出
		exportDeviceListTable(){
			var vm =this;
			exportByForm("${ctx}/task/recurringReboot/exportRecuDeviceReboot.action",{
				timeZone: timeZone,
				operatorCode:operator_code,
				searchText: vm.queryDeviceParams.searchText
			});
		},
		// 打开定时重启设置页面
		settingRecurringReboot(){
			var vm = this;
						
			vm.recurringRebootSlideUrl = "${ctx}/task/recurringReboot/toTaskConfigPage.action";
			vm.recurringRebootSlideHeight = '100%';
			vm.recurringRebootSlideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.recurringRebootSlidePosition = 'top';
			vm.recurringRebootSlideHeader = true;
			vm.recurringRebootSlideTitle = '<%=rb.getString("DingShiChongQiRenWu")%>';
			if(vm.recurringRebootSwitch == '1'){
				vm.recurringRebootSlideFooter = false;
			}else{
				vm.recurringRebootSlideFooter = true;
			}
			vm.$refs.recurringRebootSlide.showSlide(function(){
				vm.modal = false;
				eventBus.$emit('recurringRebootTask-init',vm.recurringRebootSwitch)
			});
		},
		// 定时重启设置提交
		saveRecurringRebootTask(){
			var vm = this;
			eventBus.$emit('recurringRebootTask-submit')
		},
		// 定时重启设置页面关闭触发表单校验
		cancelRecurringRebootSlide(){
			var vm = this;
			eventBus.$emit('recurringRebootTask-cancel')
		},
		//定时重启设置页面关闭事件
		closeRecurringRebootSlide(){
			var vm = this;
			vm.$refs.recurringRebootSlide.hide();
		},
		// 定时重启开关
		recurringRebootSwitchChange(val){
			var vm = this,
				params={
					switch:val,
					operatorCode:operator_code
				};
			if(val == '0'){
				vm.recurringRebootSwitch = '1'
				vm.$confirm('<%=rb.getString("QueDingTingZhiZhouQiRenWu")%>','<%=rb.getString("QueRen")%>',{
					closeOnClickModal:false,
					customClass:'warningConfirm',
					callback: function(valid){
						if(valid == 'confirm') {
							axios.post('${ctx}/task/recurringReboot/setSwitch.action',stringify(params)).then(function(response){
								var data = response.data;
								
								if(data) {
									if(data["success"]){
										vm.$message({
											message: '<%=rb.getString("ChengGong")%>',
											type:'success'
										});
										vm.recurringRebootSwitch = '0'
									}else{
										vm.$message.error(data["message"])
									}
								}
							}).catch(function(error){})
						}else{
							vm.recurringRebootSwitch = '1';
						}
					}
				});
			}else{
				axios.post('${ctx}/task/recurringReboot/setSwitch.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
						}else{
							vm.recurringRebootSwitch = '0';
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			}
		},
		// 定时重启任务 模糊查询
		queryTask(val){
			var vm = this;

			vm.resetTaskQuery();
			vm.queryTaskParams.searchText= val;
		},
		// 定时重启任务 高级查询
		taskAdvanceQuery(){
			var vm = this;
			Object.assign(vm.queryTaskParams, vm.queryTaskForm);
			if(vm.timeRange != null){
				vm.queryTaskParams.startTime = vm.timeRange[0];
				vm.queryTaskParams.endTime = vm.timeRange[1];
			}else{
				vm.queryTaskParams.startTime = '';
				vm.queryTaskParams.startTime = '';
			}
		},
		// 定时重启任务 高级查询重置
		resetTaskQuery(){
			var vm = this,
				params = {
					searchText: '',
					taskName:'',
					startTime:'',
					endTime:'',
				};
			vm.timeRange = [];
			Object.assign(vm.queryTaskForm, params);
			Object.assign(vm.queryTaskParams, params);
		},
		// 定时重启 任务和设备表格tabs切换事件
		taskAndDeviceTabChange(val){
			var vm = this;

			if(vm.taskAndDeviceShow == val)return
			vm.taskAndDeviceShow = val;
		},
		// 定时重启 任务结果 表格模糊查询
		queryDeviceList(val){
			var vm = this;
			vm.queryDeviceParams.searchText= val;
		},
		// 升级任务结果格式化
		taskResultFmt(row,column,cellValue,index){//
			var resultObj = {
					"1" : '<%=rb.getString("ChengGong")%>',
					"2" : '<%=rb.getString("BuFenChengGong")%>',
					"3" : '<%=rb.getString("ShiBai")%>',
					"" : ""
			}
			return resultObj[cellValue];
		},
		// 设备结果格式化
		deviceResultFmt(row,column,cellValue,index){//
			var resultObj = {
					"1" : '<%=rb.getString("ChengGong")%>',
					"2" : '<%=rb.getString("ZhongZhi")%>',
					"3" : '<%=rb.getString("ShiBai")%>',
					"" : ''
			}
			return resultObj[cellValue];
		},
		//点击添加按钮进入新建任务页面
	    addRebootTask(){
	    	var vm = this;
	    	if(vm.ifAddFlag){
		    	vm.createRebootTask();
	    	}else{
	    		vm.cancelSlide();
	    	}
	    },
	    saveRebootTask(){//保存新建任务
	    	eventBus.$emit('hander-ok')
	    },
	    cancelSlide(){//退出新建任务页面
	    	if(this.operateType == 'view'){
	    		this.$refs.slide.hide();
	    	}else{
	    		eventBus.$emit('hander-cancel')
	    	}
	    },
	    hideRebootTask(){
			var vm = this;
			if(vm.operateType == 'add'){
	    		vm.buttonIcon = 'el-icon-circle-add'
		    	vm.buttonText='<%=rb.getString("TianJia")%>'
		    	vm.ifAddFlag = true
			}
	    	vm.$refs.slide.hide();
	    },
	    closeRebootTask(){
			var vm = this;
	    	if(vm.operateType == 'add'){
	    		vm.buttonIcon = 'el-icon-circle-add'
		    	vm.buttonText='<%=rb.getString("TianJia")%>'
		    	vm.ifAddFlag = true
	    	}
	    	vm.$refs.slide.hide();
	    	vm.$refs.reboot.$refs.ctable.refresh()
	    },
	    showText(){
	    	this.addText = true;
	    },
	    hideText(){
	    	this.addText = false;
	    },
	    exportMsg(text){
	    	eventBus.$emit('export-result',this.rowData.TASK_ID,this.rowData.TYPE)
	    },
	    hideChoseList(){
	    	this.showList = false;
	    },
	    createRebootTask(){ //新建任务
	    	var vm = this;
			vm.styleObj = {
   	    			zIndex:10
   	    	}
    		//vm.buttonIcon = 'el-icon-circle-close';
    	    //vm.buttonText='<%=rb.getString("GuanBi")%>';
    	    vm.slideTitle='<%=rb.getString("XinJianRenWu")%>';
    	    vm.slideHeader = true;
    	    vm.slideUrl = '${ctx}/task/reboot/goAddTask.action';
    	    vm.slideFooter = 'true';
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.operateType = 'add';
    	    vm.exportFlag = false;
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
    	    vm.ifAddFlag = false;
	    }
	}
})
</script>