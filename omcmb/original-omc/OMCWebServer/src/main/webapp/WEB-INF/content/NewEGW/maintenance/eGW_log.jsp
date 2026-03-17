<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwLogPage .container .operations{
		right: 10px;
	}
	#egwLogPage .slide-position-top .el-card__body{
		box-sizing: border-box !important;
	}
	.egwLogMainBox{
		height: 100%;
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	.container .cmenu{
		z-index: 361!important;
	}
	.addTaskBtnCls .el-icon::before{
		font-size: 30px;
	}
	.deviceTable{
		height: 460px;
	}
	.flex-form {
		display: flex;
		flex-wrap: wrap;
	}
	.flex-form .el-form-item {
		margin-right: 100px;
		margin-bottom: 10px;
	}
	.egwLogTaskBox{
		box-sizing: border-box;
		flex: 1;
		background-color: #FFFFFF;
		padding-top: 20px;
		overflow: auto;
	}
	.egwLogTaskBox .el-tabs{
		height: 100%;
	}
	#egwLogPage .el-tabs__active-bar{
		bottom: unset;
		top: 0;
		padding: 0px 20px;
	}
	.el-tabs--top .el-tabs__item{
		font-size:16px;
		line-height:40px;
	}
	.el-tabs--top .el-tabs__item.is-top:nth-child(2){
		margin-left:20px;
	}
	#egwLogPage .el-tabs__header{
		background-color: #FFFFFF;
		border: 1px solid #E9E9E9;
		border-bottom: none;
	}
	#egwLogPage  .el-tabs__content{
		border-top: none;
	}
	#egwLogPage .egwLogTaskBox .el-tabs__header{
		border-top: none;
		border-bottom: 1px solid #E9E9E9;
	}
	.egwLogTaskBox .el-tabs__item{
		font-size:14px;
	}
	.egwLogTaskBox .el-tabs--top{
		border:none;
	}
	.egwLogTaskBox .el-tabs__nav-scroll{
		margin-left:10px;
	}
	.productTypeBox{
		margin-top: 10px;
		height: 40px;
		border: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
		background-color: #F6F7FB;
		padding-left: 10px;
	}
	.productTypeBox div:first-child{
		font-weight: bold;
		font-size: 12px;
		color:#333333;
		margin-right: 20px;
	}
	.productTypeBox .productTypeDefault{
		height: 24px;
		width: 60px;
		font-size: 12px;
		color: #666666;
		line-height: 24px;
		text-align: center;
		box-sizing: border-box;
		cursor: pointer;
	}
	.productTypeBox .productTypeDefault:hover{
		height: 24px;
		width: 60px;
		font-size: 12px;
		background-color: #FFFFFF;
		line-height: 24px;
		color: #4D84FF;
		text-align: center;
		border: 1px solid #4D84FF;
		box-sizing: border-box;
		border-radius: 13px;
		cursor: pointer;
	}
	.productTypeBox  .productTypeSelect{
		height: 24px;
		width: 60px;
		font-size: 12px;
		background-color: #FFFFFF;
		line-height: 24px;
		color: #4D84FF;
		text-align: center;
		border: 1px solid #4D84FF;
		box-sizing: border-box;
		border-radius: 13px;
		cursor: pointer;
	}
	.taskHeadBox{
		position: absolute;
		top:10px;
		right: 30px;
		display: flex;
		z-index: 99;
	}
	.taskHeadBox  .statisticsDiv{
		border: 1px solid #4D84FF;
		height: 30px;
		margin-left: 15px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		box-sizing: border-box;
		overflow: hidden;
	}
	.statisticsDiv > div:first-child{
		padding: 0px 10px;
		color: #4D84FF;
		height: 30px;
		line-height: 30px;
		border-radius: 4px;
		font-size: 12px;
		background-color: #F2f6ff;
	}
	.statisticsDiv > div:last-child{
		padding: 0px 20px;
		height: 30px;
		line-height: 30px;
		border-left: 1px solid #4D84FF;
	}
	.statisticsDiv .el-icon, .statisticsSuccessDiv .el-icon, .statisticsFailDiv .el-icon{
		margin-right: 10px;
	}
	.resultHeadBox{
		position: absolute;
		top:10px;
		right: 30px;
		display: flex;
		z-index: 99;
	}
	.resultHeadBox .statisticsSuccessDiv{
		border: 1px solid #67D972;
		height: 30px;
		border-radius: 4px;
		display: flex;
		margin-right: 15px;
		align-items: center;
		box-sizing: border-box;
		overflow: hidden;
	}
	.statisticsSuccessDiv .el-icon::before{
		font-size: 16px;
		color:#67D972;
	}
	.statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
		height: 30px;
		line-height: 30px;
		border-radius: 4px;
		font-size: 12px;
		background-color: #EEFFF3;
	}
	.statisticsSuccessDiv > div:last-child{
		padding: 0px 20px;
		height: 30px;
		line-height: 30px;
		border-left: 1px solid #67D972;
	}
	.resultHeadBox .statisticsFailDiv{
		border: 1px solid #E88282;
		height: 30px;
		border-radius: 4px;
		display: flex;
		align-items: center;
		margin-right: 15px;
		box-sizing: border-box;
		overflow: hidden;
	}
	.statisticsFailDiv .el-icon::before{
		font-size: 16px;
		color: #E88282;
	}
	.statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
		height: 30px;
		line-height: 30px;
		border-radius: 4px;
		font-size: 12px;
		box-sizing: border-box;
		background-color: #FEF2F2;
		
	}
	.statisticsFailDiv > div:last-child{
		padding: 0px 20px;
		height: 30px;
		line-height: 30px;
		border-left: 1px solid #E88282;
	}
</style>
<div class="pageDefault" id='egwLogPage'>
	<div class="container">
		<div class="egwLogMainBox">
			<!-- 操作按钮 -->
			<div class="operations">
				<div class="placeholder-bt" placeholder="<%=rb.getString("XinJianRenWu")%>">
					<div class="addTaskBtnCls" >
						<span class="el-icon el-icon-circle-add CODE_EGW hidden" @click="addTask"></span>
					</div>
					
				</div>
			</div>
		
			<div class="deviceTable">
				<el-ctable
					:url="deviceTableUrl"
					:query-params="queryDeviceParams" 
					ref="egwLogDeviceTable" 
					:height="height" 
					@selection-change='deviceSelect'
					:page-size="pageSize" 
					:page-list="pageList" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<div style="display: flex;align-items: center;">
							<h3 style="padding-left: 10px;">eGW List</h3>
							<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("EGWMingCheng")%> / <%=rb.getString("eGWIP")%>"></el-query>
						</div>
						
					</template>
						<!-- 列表columns -->
					<el-table-column type="selection" width="45"></el-table-column>
					<el-table-column prop="connection_status" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connection_status=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop="gw_name" show-overflow-tooltip label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
					<el-table-column prop="gw_ip" show-overflow-tooltip label="<%=rb.getString("eGWIP")%>" min-width="160" ></el-table-column>
					<el-table-column prop="gw_port" show-overflow-tooltip label="<%=rb.getString("EGWDuanKou")%>" min-width="250" ></el-table-column>
				
				</el-ctable>
			</div>
			<div class="egwLogTaskBox">
				<el-tabs v-model="activeName" type="card" >
					<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="task">
						<div class="taskHeadBox">
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%></div>
								<div>{{statisticTaskStatus.waiting}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%></div>
								<div>{{statisticTaskStatus.inProgress}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-suspend"></span><%=rb.getString("ZanTing")%></div>
								<div>{{statisticTaskStatus.suspend}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-terminate"></span><%=rb.getString("JieShu")%></div>
								<div>{{statisticTaskStatus.end}}</div>
							</div>
						</div>
						<el-ctable 
							id="egwLogTaskTable" 
							time=6  
							@load-success="taskTableLoadSuccess" 
							ref="egwLogTaskListTable" 
							:url="taskListTableUrl" 
							:query-params="queryTaskParams" 
							:page-size="20" 
							:pagination=true 
							height="100%"
						 >
							<template slot="toolbar">
								<el-query id="queryTaskTable" @query="queryTask" @advance-query="taskAdvanceQuery" @reset="resetTaskQuery" placeholder="<%=rb.getString("RenWuMingCheng")%>"
									:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
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
							<el-table-column label="" width="30" class-name="no-text-tips">
								<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
									<div class="el-icon el-icon-operation-more" @click="taskOptClick(scope.row,event)" v-clickoutside="hideMenus"></div>
								</template>
							</el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="200"></el-table-column>
							<el-table-column prop="user" label="<%=rb.getString("CaoZuoRen")%>"  min-width="100" ></el-table-column>
							<el-table-column prop="operationTime" label="<%=rb.getString("CaoZuoShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="status" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="changePasswordTaskTableStatus(scope.row.status)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="progress" label="<%=rb.getString("JinDu")%>" min-width="100"></el-table-column>
							<el-table-column prop="result" label="<%=rb.getString("JieGuo")%>" min-width="100" :formatter="taskTableResult"></el-table-column>
							<el-table-column prop="startTime" label="<%=rb.getString("KaiShiShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" min-width="180"></el-table-column>
						</el-ctable>
						<el-cmenu ref="taskMenu" :data="taskMenus" @click="taskMenuClick"></el-cmenu>
					</el-tab-pane>
					<el-tab-pane label="<%=rb.getString("SheBeiLieBiao")%>" name="device"> 
						<div class="resultHeadBox">
							<div class="statisticsSuccessDiv">
								<div><span class="el-icon el-icon-circle-success"></span><%=rb.getString("ChengGong")%></div>
								<div>{{statisticDeviceResult.success}}</div>
							</div>
							<div class="statisticsFailDiv">
								<div><span class="el-icon el-icon-circle-close"></span><%=rb.getString("ShiBai")%></div>
								<div>{{statisticDeviceResult.fail}}</div>
							</div>
							<div class="exportResultDiv">
								<div class="placeholder-bt" placeholder="<%=rb.getString("DaoChu")%>">
									<span class="el-icon el-icon-circle-export" @click="exportResultTable"></span>
								</div>
							</div>
						</div>
						<!--:url="queryResultUrl"  :data-->
						<el-ctable 
							id="egwLogResultTable"
							ref="queryResultTable" 
							:url="queryResultUrl" 
							:query-params="queryResultParams"
							@load-success="deviceTableLoadSuccess"
							height="100%"
							time=6
							:page-size="20" 
							:pagination=true>
							
							<template slot="toolbar">
								<el-query type="normal" @query="queryLogResult" placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWMingCheng")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
							</template>
							<el-table-column prop="taskId" v-if="false"></el-table-column>
							<el-table-column prop="serialNumber" show-overflow-tooltip label="<%=rb.getString("eGWBianMa")%>" min-width="160"></el-table-column>
							<el-table-column prop="cpeName" show-overflow-tooltip label="<%=rb.getString("EGWMingCheng")%>"  min-width="100"></el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="280"></el-table-column>
							<el-table-column prop="status" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="resultTableStatus(scope.row.status)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="result" label="<%=rb.getString("JieGuo")%>" min-width="110" :formatter="resultTableResult"></el-table-column>
							<el-table-column prop="failureReason" show-overflow-tooltip label="<%=rb.getString("PCILOCKShiBaiYuanYin")%>" min-width="120"></el-table-column>
							<el-table-column prop="startTime" label='<%=rb.getString("KaiShiShiJian")%>' min-width="160" ></el-table-column>
							<el-table-column prop="endTime" label='<%=rb.getString("JieShuShiJian")%>'  min-width="160"></el-table-column>
						</el-ctable>
					</el-tab-pane>
				</el-tabs>
			</div>
		</div>
		<!-- slide -->
		<el-slide ref="egwLogSlide" id="egwLogSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
			:height="slideHeight"  :width='slideWidth' @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
<script type="text/javascript">
var egwLogVue = new Vue({
	el:'#egwLogPage',
	data(){
		return {
			deviceTableUrl:"${ctx}/egw/register/queryEgwPageList.action",
			selection:[],
			
			queryDeviceParams:{
				searchText:'',
				timeZone:timeZone,
			},
			taskListTableUrl:'${ctx}/task/cpe/changepwd/getCPEChangePasswordList.action',
			activeName:'task',
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
			queryResultUrl:'${ctx}/task/cpe/changepwd/getDeviceList.action',
			queryResultParams:{
				timeZone: timeZone,
				searchText:'',
			},
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',

            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
			taskMenus:[],
            rowData:'',
			taskRowData:'',
			statisticTaskStatus:{
				waiting: 0,
				inProgress: 0,
				suspend: 0,
				end: 0
			},
			statisticDeviceResult:{
				fail: 0,
				success: 0
			},
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
    },
	watch:{},
	methods:{
		// 任务表格 加载成功回调
		taskTableLoadSuccess(data){
			var vm = this;

			Object.assign(vm.statisticTaskStatus, data.properties);
			
		},
		// 设备表格 加载成功回调
		deviceTableLoadSuccess(data){
			var vm = this;

			Object.assign(vm.statisticDeviceResult, data.properties);
		},
		// 新建设备升级任务
		addTask(){
			var vm = this;
			vm.slideHeader = false;
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWLogAddTask.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("XinJianMiMaXiuGaiRenWu")%>';
			
			vm.$refs.egwLogSlide.showSlide(()=>{
				eventBus.$emit('action-init','','add');
			})
		},
		// 设备表格选择事件
		deviceSelect(selection){
			var vm = this;

			vm.selection = selection
		},
		// 结果 导出
		exportResultTable(){
			var vm = this;

			exportByForm("${ctx}/task/cpe/changepwd/exportDeviceListToCSV.action",{
				timeZone: timeZone,
				searchText: vm.queryResultParams.searchText
			});
		},
		// EGW 重启任务结果 表格模糊查询
		queryLogResult(val){
			var vm = this;
			vm.queryResultParams.searchText= val;
		},
		/**
		 *  任务结果转换
		 * @param cellValue:传入的 结果 数据 进行转换
		*/
		taskTableResult(row,column,cellValue,index){
			var resultObj = {
				"1" : "<%=rb.getString("ChengGong")%>",
				"2" : "<%=rb.getString("BuFenChengGong")%>",
				"3" : "<%=rb.getString("ShiBai")%>",
				"" : "",
			}
			return resultObj[cellValue];
		},
		// 设备结果格式化
		resultTableResult(row,column,cellValue,index){
			var resultObj = {
					"1" : '<%=rb.getString("ChengGong")%>',
					"2" : '<%=rb.getString("ZhongZhi")%>',
					"3" : '<%=rb.getString("ShiBai")%>',
					"" : ''
			}
			return resultObj[cellValue];
		},
		// 设备表格 模糊查询
		queryDevice(val){
			var vm = this;
			vm.queryDeviceParams.searchText= val;
		},
		// 升级任务表格 模糊查询
		queryTask(val){
			var vm = this;

			vm.resetTaskQuery();
			vm.queryTaskParams.searchText= val;
		},
		// 任务表格 高级查询
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
		// 任务表格 高级查询重置
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
		// 任务表格 打开操作菜单
		taskOptClick(row,evt){
			var vm = this;

			vm.taskRowData = row;
			var status = row.status;
			vm.taskMenus = [
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_EGW hidden",code:'start'},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_EGW hidden",code:'wait'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_EGW hidden",code:'end'},
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_EGW hidden",code:'del'},
            ];
			initTaskStatus(status,vm.taskMenus);
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.taskMenu.show(evt)
            })
		},
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
		taskMenuClick(evt){
			var vm = this;
			var codes = {
				start:this.taskStart,	// 开始
				wait:this.taskStop,	// 暂停
				end:this.taskTerminate,	// 终止
	    		info:this.taskInfo,  // 详情
	    		mod:this.taskModify,	// 修改
	    		del:this.taskDel,	// 删除
	    	};
			if(codes[evt.code]){
				codes[evt.code]();
			}
		},
		/**
		 * 激活任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		taskStart(id,type){
			var vm = this;
			axios.post('${ctx}/task/cpe/changepwd/activeCPEChangePassword.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwLogTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 暂停任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		taskStop(){
			var vm = this;
			axios.post('${ctx}/task/cpe/changepwd/suspendCPEChangePassword.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwLogTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 终止任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		taskTerminate(){
			var vm = this;
			axios.post('${ctx}/task/cpe/changepwd/terminateCPEChangePassword.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwLogTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 查看任务
		 * @param id:当前数据id
		*/
		taskInfo(){
			var vm = this;
			vm.slideHeader = false;
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWLogAddTask.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.egwLogSlide.showSlide(()=>{
				eventBus.$emit('action-init',vm.taskRowData.taskId,'view');
			})
		},
		/**
		 * 修改任务
		 * @param id:当前数据id
		*/
		taskModify(id){
			var vm = this;
			vm.slideHeader = false;
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWLogAddTask.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.egwLogSlide.showSlide(()=>{
				eventBus.$emit('action-init',vm.taskRowData.taskId,'edit');
			})
		},
		/**
		 * 删除升级任务
		 * @param id:当前数据id
		*/
		taskDel(){
			var vm = this,
				params = {
					taskId: vm.taskRowData.taskId,
				};
			vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(function(){
				axios.post('${ctx}/task/cpe/changepwd/deleteCPEChangePassword.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.egwLogTaskListTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		// 菜单关闭
        hideMenus() {
			this.$refs.taskMenu.hide();
        },
		// slide 提交
		submitSlide(){
			var vm = this;
			eventBus.$emit('action-save');
		},
		// 直接关闭slide事件
        hideSlide(){
			var vm = this;
			vm.$refs.egwLogSlide.hide();
			vm.$refs.egwLogTaskListTable.refresh();
			vm.$refs.queryResultTable.refresh();
			vm.$refs.egwLogDeviceTable.clearSelection();
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			eventBus.$emit('cancel-slide');
            
		},
	},
	mounted(){
		eventBus.$off('hide-egwLog-slide').$on('hide-egwLog-slide',this.hideSlide);
	}
	
})

</script> 
