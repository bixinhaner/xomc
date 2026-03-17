<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	
	#egwRebootPage .container .cmenu{
		z-index: 361!important;
	}
	#egwRebootPage .deviceTable{
		height: 50%;
	}

	#egwRebootPage .egwRebootTaskBox{
		box-sizing: border-box;
		flex: 1;
		background-color: #FFFFFF;
		overflow: auto;
		margin-top: 10px;
	}
	#egwRebootPage .taskHeadBox{
		position: absolute;
		top:15px;
		right: 0px;
		display: flex;
		z-index: 99;
	}
	#egwRebootPage .taskHeadBox .statisticsDiv{
		height: 16px;
		line-height: 16px;
		display: flex;
		overflow: hidden;
		font-size: 14px;
	}
	#egwRebootPage .statisticsDiv > div:first-child{
		padding: 0px 0 0 10px;
		color: #4D84FF;
		background: #FFFFFF;
		height: 16px;
		line-height: 16px;
	}
	#egwRebootPage .statisticsDiv > div:last-child{
		padding: 0px 10px;
	}
	#egwRebootPage .statisticsDiv .el-icon, #egwRebootPage .statisticsSuccessDiv .el-icon, #egwRebootPage .statisticsFailDiv .el-icon{
		margin-right: 6px;
		font-size: 14px;
	}
	#egwRebootPage .resultHeadBox{
		position: absolute;
		top:17px;
		right: 60px;
		display: flex;
		z-index: 99;
	}
	#egwRebootPage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#egwRebootPage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#egwRebootPage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#egwRebootPage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#egwRebootPage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#egwRebootPage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#egwRebootPage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}
</style>
<!--egw重启 -->
<div class="overflow-cls">
<div class="pageDefault" id='egwRebootPage' style="min-width: 1200px; width: 100%;position: relative; border: 0; background: #F6F7FB;">
	<div class="container ">
		<div class="egwRebootMainBox ">
			<!-- 操作按钮 -->
			<div class="operations">
				<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:20px;top:3px;" @click="addTask" tip="<%=rb.getString("XinJianRenWu")%>">
					<span class='el-icon el-icon-plus'></span>
				</div>
			</div>
		
			<div class="deviceTable commonWarp" style='height: calc(50% - 5px)'>
				<el-ctable
					:url="deviceTableUrl"
					:query-params="queryDeviceParams" 
					ref="egwRebootDeviceTable" 
					:height="height" 
					@selection-change='deviceSelect'
					:page-size="pageSize" 
					:page-list="pageList" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<div class='commonQuery commonFlex'>
							<span class="commonText14" style='padding: 4px 0 0 10px'>WCG List</span>
							<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWIP")%>"></el-query>
						</div>
						
					</template>
						<!-- 列表columns -->
					<el-table-column v-if="optBtnShow" type="selection" width="45"></el-table-column>
					<el-table-column prop="connectionStatus" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connectionStatus!='Exception' && scope.row.connectionStatus!='On' && scope.row.connectionStatus!='updating' && scope.row.connectionStatus!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connectionStatus=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connectionStatus=='On'||scope.row.connectionStatus=='updating'||scope.row.connectionStatus==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
                	<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
					<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
					<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
					<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
				
				</el-ctable>
			</div>
			<div class="egwRebootTaskBox commonWarp" style='height: calc(50% - 5px)'>
				<el-tabs v-model="activeName" class=" newTabs" style='height: 100%;'>
					<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="task">
						<div class="taskHeadBox">
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%></div>
								<div>{{statisticTaskStatus.waitingNum}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%></div>
								<div>{{statisticTaskStatus.processingNum}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-suspend"></span><%=rb.getString("ZanTing")%></div>
								<div>{{statisticTaskStatus.suspendedNum}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-terminate"></span><%=rb.getString("JieShu")%></div>
								<div>{{statisticTaskStatus.endNum}}</div>
							</div>
						</div>
						<el-ctable 
							id="egwRebootTaskTable" 
							time=6  
							@load-success="taskTableLoadSuccess" 
							ref="egwRebootTaskListTable" 
							:url="taskListTableUrl" 
							:query-params="queryTaskParams" 
							:page-size="20" 
							:pagination=true 
							height="100%">
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style=" padding: 0 0 10px 0;">
									<el-query type="normal" @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
									<el-date-picker style='margin-left: 10px;' 
										v-model="dateValue"
										type="datetimerange"
										value-format="yyyy-MM-dd HH:mm:ss"
										range-separator="——"  
										@change="dateChange"
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
										end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
									</el-date-picker>
								</div>
							</template>
							<el-table-column label="" width="30" class-name="no-text-tips">
								<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
									<div class="el-icon el-icon-operation-more" @click="taskOptClick(scope.row,event)" v-clickoutside="hideMenus"></div>
								</template>
							</el-table-column>
							<el-table-column prop="taskId"  v-if="false"></el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="200"></el-table-column>
							<el-table-column prop="createUser" label="<%=rb.getString("CaoZuoRen")%>"  min-width="100" ></el-table-column>
							<el-table-column prop="createTime" label="<%=rb.getString("CaoZuoShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="taskStatus" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="taskTableStatus(scope.row.taskStatus)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="taskProgress" label="<%=rb.getString("JinDu")%>" min-width="100"></el-table-column>
							<el-table-column prop="taskResult" label="<%=rb.getString("JieGuo")%>" min-width="100" :formatter="taskTableResult"></el-table-column>
							<el-table-column prop="startTime" label="<%=rb.getString("KaiShiShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" min-width="180"></el-table-column>
						</el-ctable>
						<el-cmenu ref="taskMenu" :data="taskMenus" @click="taskMenuClick"></el-cmenu>
					</el-tab-pane>
					<el-tab-pane label="<%=rb.getString("SheBeiLieBiao")%>" name="device"> 
						<div class="resultHeadBox">
							<div class="statisticsSuccessDiv">
								<div><span class="el-icon el-icon-circle-success"></span><%=rb.getString("ChengGong")%></div>
								<div>{{statisticDeviceResult.sucNum}}</div>
							</div>
							<div class="statisticsFailDiv">
								<div><span class="el-icon el-icon-circle-close"></span><%=rb.getString("ShiBai")%></div>
								<div>{{statisticDeviceResult.failNum}}</div>
							</div>
						</div>
						<div class="newIconBoxCls-bt" style="right: 10px; top: 10px;" @click="exportResultTable" tip="<%=rb.getString("DaoChu")%>">
							<span class='el-icon el-icon-circle-export'></span>
						</div>
						<!--:url="queryResultUrl"  :data-->
						<el-ctable 
							id="egwRebootResultTable"
							ref="queryResultTable" 
							:url="queryResultUrl" 
							:query-params="queryResultParams"
							@load-success="deviceTableLoadSuccess"
							height="100%"
							time=6
							:page-size="20" 
							:pagination=true>
							
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style="padding: 0 0 10px 0;">
								<el-query type="normal" @query="queryRebootResult" placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWMingCheng")%> / <%=rb.getString("EGWIP")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
								</div>
							</template>
							<el-table-column prop="ID" v-if="false"></el-table-column>
							<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
							<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
							<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="280"></el-table-column>
							<el-table-column prop="progressStatus" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="resultTableStatus(scope.row.progressStatus)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="progressResult" label="<%=rb.getString("JieGuo")%>" min-width="110" :formatter="resultTableResult"></el-table-column>
							<el-table-column prop="failureReason" show-overflow-tooltip label="<%=rb.getString("PCILOCKShiBaiYuanYin")%>" min-width="120"></el-table-column>
							<el-table-column prop="startTime" label='<%=rb.getString("KaiShiShiJian")%>' min-width="160" ></el-table-column>
							<el-table-column prop="endTime" label='<%=rb.getString("JieShuShiJian")%>'  min-width="160"></el-table-column>
						</el-ctable>
					</el-tab-pane>
				</el-tabs>
			</div>
		</div>
		<!-- slide -->
		<el-slide ref="egwRebootSlide" id="egwRebootSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" class='commonBorderSlide'
			:height="slideHeight"  :width='slideWidth' :subloading="slideSubmitLoading" @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
</div>
<script type="text/javascript">
var egwRebootVue = new Vue({
	el:'#egwRebootPage',
	data(){
		return {
			deviceTableUrl:"${ctx}/egw/monitor/getEgwMonitorPageList.action",
			selection:[],
			
			queryDeviceParams:{
				search_text:'',
				timeZone:timeZone,
			},
			taskListTableUrl:'${ctx}/egw/reboot/getRebootTaskList.action',
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
			queryResultUrl:'${ctx}/egw/reboot/getRebootTaskDevices.action',
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
            slideSubmitLoading:'',

            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
			taskMenus:[],
            rowData:'',
			taskRowData:'',
			statisticTaskStatus:{
				waitingNum: 0,
				processingNum: 0,
				suspendedNum: 0,
				endNum: 0
			},
			statisticDeviceResult:{
				failNum: 0,
				sucNum: 0
			},
			dateValue:[],
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		optBtnShow() {
			return writableMap['CODE_EGW'] == true;
		},
    },
	watch:{
		dateValue(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.queryTaskParams.startTime = '';
    			vm.queryTaskParams.endTime = '';
			}
		}
	},
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
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWRebootAddTask.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinJianMiMaXiuGaiRenWu")%>';
			
			vm.$refs.egwRebootSlide.showSlide(()=>{
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

			exportByForm("${ctx}/egw/reboot/exportTaskDevices.action",{
				timeZone: timeZone,
				searchText: vm.queryResultParams.searchText
			});
		},
		// EGW 重启任务结果 表格模糊查询
		queryRebootResult(val){
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
			vm.queryDeviceParams.search_text= val;
		},
		// 升级任务表格 模糊查询
		queryTask(val){
			var vm = this;

			vm.queryTaskParams.searchText= val;
		},
		
		// 任务表格 打开操作菜单
		taskOptClick(row,evt){
			var vm = this;

			vm.taskRowData = row;
			var status = row.taskStatus;
			vm.taskMenus = [
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_EGW hidden",code:'start'},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_EGW hidden",code:'wait'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_EGW hidden",code:'end'},
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				// {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'mod'},
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
			axios.post('${ctx}/egw/reboot/activeTask.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwRebootTaskListTable.refresh()
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
			axios.post('${ctx}/egw/reboot/suspendTask.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwRebootTaskListTable.refresh()
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
			axios.post('${ctx}/egw/reboot/terminateRebootTask.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwRebootTaskListTable.refresh()
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
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWRebootAddTask.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.egwRebootSlide.showSlide(()=>{
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
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWRebootAddTask.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.egwRebootSlide.showSlide(()=>{
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
				axios.post('${ctx}/egw/reboot/delRebootTask.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.egwRebootTaskListTable.refresh()
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
			vm.$refs.egwRebootSlide.hide();
			vm.$refs.egwRebootTaskListTable.refresh();
			vm.$refs.queryResultTable.refresh();
			vm.$refs.egwRebootDeviceTable.clearSelection();
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			eventBus.$emit('cancel-slide');
            
		},
		dateChange(val) {
			var vm = this;
			vm.dateValue = val;
			if(val != null){
				vm.queryTaskParams.startTime = vm.dateValue[0];
				vm.queryTaskParams.endTime = vm.dateValue[1];
			}
		},
	},
	mounted(){
		eventBus.$off('hide-egwReboot-slide').$on('hide-egwReboot-slide',this.hideSlide);
	}
	
})

</script> 
