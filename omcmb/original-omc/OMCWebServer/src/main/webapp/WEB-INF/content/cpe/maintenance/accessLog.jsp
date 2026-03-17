<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>

<style>
	#cpeLogs .deviceLogBox{
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	#cpeLogs .deviceLogBox .deviceListBox{
		height: 50%;
		border-bottom: 1px solid #E9E9E9
	}
	#cpeLogs .deviceLogBox .logFileBox{
		height: 50%;
	}
	#cpeLogs .logFileBox .logFileTitle{
		font-size: 14px;
		padding-left: 20px;
		font-weight: 550;
		color:#333333;
	}
	#cpeLogs .el-icon-status-terminate:before,
	#cpeLogs .el-icon-status-waiting1:before,
	#cpeLogs .el-icon-status-inProgress:before{
		color:#4D84FF;
	}
	#cpeLogs .el-icon-status-success:before {
		color:#67D972;
	}
	
	.cpelogWenJianBox{
		height: 60%;
		display: flex;
		overflow: visible;
		border: 1px solid #d1ecf5;
	}
	.cpelogWenJianBoxPad{
		width: 260px;
		border-right: 1px solid #d1ecf5;
	}
</style>
<div class="overflow-cls">
<!-- 网管日志功能页面 -->
<div class="panelDefault" id='cpeLogs' style="min-width: 1000px;position: relative; overflow: hidden;">
	
	<!-- 主页面区域 -- tab页 -->
	<el-tabs v-model="activeName" @tab-click='tabClick' style="height: 100%;">
		<!-- 设备日志 -->
		<el-tab-pane label="<%=rb.getString("DeviceSheBeiRiZhi")%>" name="first">
			<!-- 按钮  -- 新增 -->
			<div v-if="hasLogRole" class="circleIcon" style="right:10px;">
				<span class="el-icon el-icon-circle-add" @click="addLogTask"></span>
				<div class="titleButtonText"><%=rb.getString("XinZeng")%></div>
			</div>
			<div class="deviceLogBox">
				<div class="deviceListBox">
					<el-ctable 
						id="deviceListTable" 
						ref="deviceListTable" 
						:time="6" 
						:url="deviceListUrl" 
						:height="height" 
						:limit="20"
						@row-click="deviceRowClick"
						@load-success="tableLoadSuccess" 
						:row-key="'device_code'" 
						:query-params="queryDeviceParams" 
						pagination="true" 
						:rownumber=true 
						@selection-change='deviceSelect'>
						<!-- 高级查询 -- 设备上报日志-->
						<template slot="toolbar">
							<el-form :model="params_device" ref="params_device" label-position="top" inline=true>
								<el-query @query="queryDevice" @advance-query="advanceQueryDevice" @reset="resetQueryDevice" :ok-text="'<%=rb.getString("ChaXun")%>'" 
									placeholder="<%=rb.getString("CPEBianMa")%>/<%=rb.getString("CPEName")%>/<%=rb.getString("CPEMacAddress")%>/<%=rb.getString("IMSI")%>" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'" :arrow-text="'<%=rb.getString("GaoJiChaXun")%>'">
									<template slot="form">
										<el-form-item label='<%=rb.getString("CPEBianMa")%>' prop="search_text">
											<el-input v-model="params_device.serial_number"></el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("CPEName")%>' prop="search_text">
											<el-input v-model="params_device.cpe_name"></el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("CPEMacAddress")%>' prop="search_text">
											<el-input v-model="params_device.mac_address"></el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("IMSI")%>' prop="search_text">
											<el-input v-model="params_device.imsi"></el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("CaoZuoShiJian")%>'>
											<el-date-picker type="datetimerange" v-model="timeRange" value-format="yyyy-MM-dd HH:mm:ss" 
													start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
										</el-form-item>
									</template>
								</el-query>
							</el-form>
						</template>
						
						<el-table-column label='' width="50" type="selection"></el-table-column>
						<!-- 主列表 -->
						<el-table-column v-if="hasLogRole" label='' width="30" prop="" class-name="no-text-tips">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more curpo" @click="deviceOptClick(scope.row,event)" v-clickoutside="handerClose"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("CPEBianMa")%>' min-width="200" prop="serial_number"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="160" ></el-table-column>
						<el-table-column prop="cpe_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="240" ></el-table-column>
						<el-table-column prop="macaddress" label="<%=rb.getString("CPEMacAddress")%>" min-width="160" ></el-table-column>
						<el-table-column prop="task_status" label="<%=rb.getString("ShouJiZhuangTai")%>" min-width="220">
							<template slot-scope="scope">
								<div v-html="cpeLogTaskTableStatus(scope.row.task_status)"></div>
							</template>
						</el-table-column>	
						<!--<el-table-column label='<%=rb.getString("WenJianShuLiang")%>' min-width="120" prop="file_num" ></el-table-column>-->
						<el-table-column prop="task_status" label="<%=rb.getString("JieGuo")%>" min-width="100" >
							<template slot-scope="scope">
								<div v-if="scope.row.task_status == '2'">
									<span><%=rb.getString("ChengGong")%></span>
								</div>
								<div v-if="scope.row.task_status == '3'">
									<span><%=rb.getString("ShiBai")%></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="180" prop="failureReason" show-overflow-tooltip="true"></el-table-column>
						<el-table-column label='<%=rb.getString("LogsGengXinShiJian")%>' min-width="150" prop="update_time"></el-table-column>

					</el-ctable>
					<el-cmenu ref="deviceMenu" :data="deviceMenus" @click="clickDeviceMenues"></el-cmenu>
				</div>
				<div class="logFileBox">
					<el-ctable id="logFileTable" ref="logFileTable" :time="6" :url="logFilelUrl" :height="height"
							:query-params="params_logFile"   page-size="10" pagination="true" :rownumber=true>
						<!-- 头部 -->
						<template slot="toolbar">
							<div class="logFileTitle">
								<%=rb.getString("WenJian")%>
								<span style="color:#4D84FF;">(MAC:{{selectLogFileMAC}})</span>
							</div>
						</template>
						<!-- 主列表 -->
						<el-table-column label='' width="30" prop="" class-name="no-text-tips">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more curpo" @click="logFileOptClick(scope.row,event)" v-clickoutside="handerClose"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="200"  prop="file_name" ></el-table-column>
						<el-table-column label='<%=rb.getString("ShouJiShiJian")%>' min-width="150" prop="upload_time" ></el-table-column>
						
					</el-ctable>
					<el-cmenu ref="logFileMenu" :data="logFileMenus" @click="clickLogFileMenues"></el-cmenu>
				</div>
			</div>
		</el-tab-pane>
		<!-- 接入日志 -->
		<el-tab-pane label="<%=rb.getString("JieRuRiZhi")%>" name="second" class=''>
			<!-- 按钮  -- 导出 -->
			<div class="circleIcon" style="right:10px;">
				<span class="el-icon el-icon-circle-export" @click="exportLogs"></span>
				<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
			</div>
			<el-ctable ref="ctableAccess" url="${ctx}/cell/cpe/log/accessLogList.action" :height="height" 
					   :query-params="params_accessLog" pagination="true" :rownumber="true">
				
				<!-- 查询 -- 接入日志 -->
				<template slot="toolbar">
					<el-query type="normal" @query="queryOp" placeholder="<%=rb.getString("IMSI")%> / <%=rb.getString("IPDiZhi")%>"></el-query>
				</template>
				
				<!-- 主列表 -->
				<el-table-column label='<%=rb.getString("IMSI")%>' min-width="135" prop="imsi"></el-table-column>
				<el-table-column label='<%=rb.getString("CPEBianMa")%>' min-width="150" prop="serialNumber" ></el-table-column>
				<el-table-column label='<%=rb.getString("CPEName")%>' min-width="130" prop="deviceName" ></el-table-column>
				<el-table-column label='<%=rb.getString("IPDiZhi")%>' min-width="100" prop="deviceIp"></el-table-column>
				<el-table-column label='<%=rb.getString("RiQi")%>' width="200" prop="createTime"></el-table-column>
			</el-ctable>
		</el-tab-pane>
	</el-tabs>
	<el-dialog title="<%=rb.getString("WenJianXinXi")%>" :visible.sync="viewFileDialogVisible" style="width:100%;" :close-on-click-modal="false" @close="cancelDialog">
		<div class="cpelogWenJianBox">
			<div class="cpelogWenJianBoxPad">
				<el-ctable id="logFileList" ref="fileList" :url="logFileListUrl" :height="'100%'" @row-click="fileSelectChange" :pagination="false"
						:query-params="params_filelist">
					<!-- 主列表 -->
					<el-table-column label='<%=rb.getString("WenJianLieBiao")%>' prop="un_file_name" ></el-table-column>
				</el-ctable>
			</div>
			<div style="width: 100%;padding: 10px;">
				<textarea id="immediateCollectFileContent" class="border border-box" style="padding-left:10px;border-style: none;width: 100%;height: 97%;resize: none;"></textarea>
			</div>
		</div>
	</el-dialog>
	 <!-- 批量操作浮层 -->
    <el-bulk ref="bulk" target="deviceListTable" :list="selectionList" row-key="device_code"
      :message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("CPEBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel: '<%=rb.getString("QuXiao")%>'}">
      <template slot="button">
        <a v-if="hasLogRole" class="linkbutton linkbutton_nowanna" @click="delCollectFile('','bulk')"><span><%=rb.getString("PiLiangQingChu")%></span></a>
        <a class="linkbutton linkbutton_trend" @click="downlodFile('','bulk')"><span><%=rb.getString("XiaZai")%></span></a>
      </template>
    </el-bulk>
	<!-- slide -->
	<el-slide ref="addCpeLogSlide" id="addCpeLogSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
		:height="slideHeight"  :width='slideWidth' :subloading="slideSubmitLoading" @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<form id = "exportLog" style="display:none" method="post"></form>
</div>
</div>
<script type="text/javascript">

var cpeLogs = new Vue({
	el:'#cpeLogs',
	data:{
		activeName:'first',
		params_accessLog:{
			timeZone:timeZone,
			searchText:'',
			likefields:'imsi,deviceIp',
			rd:''
		},
		queryDeviceParams:{
			device_type:'CPE',
			device_code:'',
			search_text:'',			
			like_fields:'serial_number',
			timeZone:timeZone,
			serial_number:'',
			cpe_name:'',
			mac_address:'',
			imsi:'',
			start_time:'',
			end_time:'',
		},
		params_device:{
			serial_number:'',
			cpe_name:'',
			mac_address:'',
			imsi:''
		},
		params_logFile:{
			timeZone:timeZone,
			taskId:'',
			deviceType:'CPE'
		},
		logFilelUrl:'${ctx}/cell/collect/getImmediateCollectLogFileDataList.action',
		selectLogFileMAC:'',
		logFileMenus:[],
		timeRange:[],
	    height:'100%',
	    width:'100%',
		deviceMenus:[],
		deviceListUrl:'${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action',
		rowDeviceData:'',
		rowFileData:'',
		viewFileDialogVisible:false,
		params_filelist: {
			taskId: '',
			fileName: '',
			fileType: 'CPE'
		},
		logFileListUrl: '${ctx}/cell/collect/doUnZipImmedLogFile.action',
		slideUrl:'',
		slideTitle:'',
		slideHeader:'',
		slideFooter:'',
		slidePosition:'',
		slideHeight:'',
		slideWidth:'',
        slideSubmitLoading:'',
		rowClickData:'',
		selectionList:[],
	},
	computed: {
		hasLogRole() {
			return writableMap.CODE_CPE_LOGS == true;
		}
	},
	watch:{
		"rowClickData":function(newVal){
			
			if(newVal){
				this.params_logFile.taskId = newVal.task_id;
				this.selectLogFileMAC = newVal.macaddress;
			}else{
				this.params_logFile.taskId = ''
			}
		},
	},
	methods:{
		init(){    //根据权限判断页面默认显示的tab内容 
			var vm = this;

		},
		// 设备表格加载成功回调
		tableLoadSuccess(data){
			var vm = this,tableData = data.rows;
			if(tableData.length > 0){
				if(vm.rowClickData){
					let curIndex = tableData.findIndex(item =>item.serial_number === vm.rowClickData.serial_number);
					if(curIndex === -1){
						vm.$nextTick(function(){
							vm.$refs.deviceListTable.setCurrentRow(tableData[0]);
							vm.rowClickData =  tableData.length > 0 ? tableData[0] : '';
						});
					}else{
						vm.$nextTick(function(){
							let rows = vm.$refs.deviceListTable.$el.querySelectorAll('tbody > tr.el-table__row');
							if(rows.length){
								rowws = Array.from(rows);
								rows.forEach(item => item.classList.remove('current-row'));
								rows[curIndex].classList.add('current-row')
							}
						});
					}
				}else{
					vm.$nextTick(function(){
						vm.$refs.deviceListTable.setCurrentRow(tableData[0]);
						vm.rowClickData =  tableData.length > 0 ? tableData[0] : '';
					});
				}
			}
			
			
		},
		// 设备单行点击事件
		deviceRowClick(row){
			var vm = this;
			vm.rowClickData = row;
		},
		//设备上报日志-模糊查询
		queryDevice(val){
			var vm = this;
			vm.resetQueryDevice();
			vm.queryDeviceParams.search_text  = val;
		},
		//设备上报日志-高级查询
		advanceQueryDevice(){
			var vm = this;
			Object.assign(vm.queryDeviceParams, vm.params_device);
			if(vm.timeRange != null){
				vm.queryDeviceParams.start_time = vm.timeRange[0];
				vm.queryDeviceParams.end_time = vm.timeRange[1];
			}else{
				vm.queryDeviceParams.start_time = '';
				vm.queryDeviceParams.end_time = '';
			}
		},
		
		//设备上报日志-重置
		resetQueryDevice(){
			var vm = this,
				params = {
					search_text: '',
					serial_number:'',
					cpe_name:'',
					mac_address:'',
					imsi:'',
					start_time:'',
					end_time:'',

				};
			vm.timeRange = [];
			Object.assign(vm.params_device, params);
			Object.assign(vm.queryDeviceParams, params);
		},
		handerClose(){ //点击页面其他地方菜单收起
			var vm = this;
	        this.$refs.deviceMenu.hide();
	        this.$refs.logFileMenu.hide();
	    },
		exportLogs(){     //导出日志数据 
			var vm = this;
			var activeName = this.$root.activeName
			if(activeName == 'second'){ 
	       	 	exportByForm('${ctx}/cell/cpe/log/exportAccessLogToCSV.action',{
	       	 		timeZone: timeZone,
	       	 		searchText: vm.params_accessLog.searchText,
	       	 		likeFields: 'imsi,deviceIp',
	       	 		content: 'imsi,serialNumber,deviceName,deviceIp,createTime'
	       	 	});
	    	}
		},
		// 设备选择
		deviceSelect(selection){
			var vm = this;
			vm.selectionList = selection;
		},
		//
		queryOp(val){   //模糊查询
			var vm = this;
		
			vm.params_accessLog.searchText  = val;
			vm.params_accessLog.rd = Math.random();
		},
	    tabClick(tab){
			var vm = this;
		},
		// 新增日志收集
		addLogTask(){
			var vm = this;
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/cell/collect/toAddTaskCPELogPage.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
			vm.$refs.addCpeLogSlide.showSlide(()=>{})
		},
		/**
		 * 更多操作
		 * @parame row:1.查看  2.终止  3.下载  4.删除 
		*/
		deviceOptClick(row,ev){ 
			var vm = this,
	    		status = row.task_status;  
	    	//0-收集未开始；1-正在收集；2-收集完成；3-收集失败；4-收集终止;5-等待上传； 
	
	    	vm.rowDeviceData = row;
			var downloadFlag = false , terminateFlag = false , showFlag = false , delFlag = false;
   
	    	if(status == 0 || status == 1 ){
	    		terminateFlag = true;
	    	}else{
	    		terminateFlag = false;
	    	}

	    	if (row.file_num > 0){
	    		downloadFlag = true
	    	}else{
	    		downloadFlag = false
	    	}
	    	
	    	vm.deviceMenus= [
		          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"el-icon el-icon-operation-terminate CODE_CPE_LOGS hidden" ,code:'terminate',disable:!terminateFlag },
		        //   {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" ,code:'download',disable:!downloadFlag},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_CPE_LOGS hidden",code:'del'}
		    ]
	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.deviceMenu.show(ev);
	    	});
	    },
		/**
		 *  获取点击项数据
		 * @parame ev:点击属性数据
		*/
	    clickDeviceMenues(ev){ //单点击方法 -- 设备上报日志
	    	var vm = this,
				codes = {
					terminate:this.terminateCollectTask,
					download:this.downlodFile,
					del:this.delCollectFile
				};
	    	if(codes[ev.code]){
	    		codes[ev.code](vm.rowDeviceData,'device')
	    	}
	    },
		/**
		 *  //终止日志收集  -- 设备上报日志、告警日志
		 * @param id:当前数据id
		 * @param status:0-收集未开始；1-正在收集；2-收集完成；3-收集失败；4-收集终止;5-等待上传；
	    	//表格处理 0-等待， 1-进行中，2-成功，3-失败，4-终止，5-正在停止上报（进行中），6- 停止上报 成功（终止），7-停止上报失败（进行中），8-重启(等待)
		*/
	    terminateCollectTask(row){  
	    	var vm = this ,
				url = '${ctx}/cell/collect/goTerminateImmediateCollectLogFile.action',
	    		params = {
					taskIds:row.task_id,
					device_code:vm.rowDeviceData.device_code,
					execute_type:vm.rowDeviceData.execute_type,
					device_type:'CPE'
				}
			var status = row.task_status;
	    	if (status != 0 && status != 1) {
	    		showMsg('prompt_msg','<%=rb.getString("MeiYouKeTingZhiShouJiDeSheBei")%>');
	            return;
	        }
	    	axios.post(url,stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.$refs.deviceListTable.refresh()
	    			vm.$message({
		    			type:'success',
		    			message:'<%=rb.getString("ChengGong")%>'
		    		})
					vm.$refs.logFileTable.refresh();
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	}).catch(function(error){
	    		
	    	})
	    },
		/**
		 *  下载文件
		 * @param row:当前数据
		*/
	    downlodFile(row,type){ 
	    	var vm = this;
				url='${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action',
	    		params = {
					timeZone: timeZone,
					taskIds:'',
					fileName: '',
					fileType: 'CPE'
				};
			if(type == 'device'){
				params.taskIds = row.task_id;
				if (row.file_num == 0) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
					return;
	        	}  
			}else if(type == 'file'){
				params.taskIds = row.task_id;
				params.fileName = row.file_name;
				if (row.file_name == null || !row.file_name || row.file_name == undefined) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
					return;
	        	}  
			}else if(type == 'bulk'){
				var taskIds=[],fileNum = 0;
				vm.selectionList.map((item,index) => {
					taskIds.push(item.task_id);
					fileNum+= parseInt(item.file_num);
				});
				if (fileNum == 0) {
					showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
					return;
	        	}  
				params.taskIds = taskIds.join(',')
			}
	        //如果没有日志文件，提示没有文件
	       
	        
			axios.post('${ctx}/cell/collect/getDownloadFileNumber.action',stringify(params)).then(function(response){
	    		var data = response.data;
				if (data.length > 0 || data.fileNum > 0) {
		    		vm.createForm(url,params);
		        } else {
		       	 	vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>') //错误提示信息
		        }
	    	}) 
   
	    },
		/**
		 * 删除任务  -- 设备上报日志 、告警日志 
		 * @param row:当前数据
		*/
	    delCollectFile(row,type){
	    	var vm = this, 
				url='${ctx}/cell/collect/doClearImmediateCollectLogFile.action',
				params={
					taskIds:'',
					fileName:'',
					deviceType:'CPE'
				};
			
			if(type == 'device'){
				params.taskIds = row.task_id;
			}else if(type == 'file'){
				params.taskIds = row.task_id;
				params.fileName = row.file_name;
			}else if(type == 'bulk'){
				var taskIds=[];
				vm.selectionList.map((item,index) => {
					taskIds.push(item.task_id)
				});
				params.taskIds = taskIds.join(',')
			}
	    	vm.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.$refs.deviceListTable.refresh()
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
			    		vm.$refs.logFileTable.refresh();
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
		/**
		 * 跳转from表格
		 * @param url:地址
		 * @param params：跳转参数
		*/
		createForm(url,param) {
			var vm = this, 
				body = document.querySelector('body'),
				form = document.createElement('form'),
				params = param || {};
			
			form.style.display = 'none';
			form.action = url;
			form.method = 'post';
			
			if(params) {
				params.token = omctoken;
				for(var key in params) {
					var input = document.createElement('input');
					input.value = params[key];
					input.setAttribute('name',key);
					form.appendChild(input);
				}
			}
			
			body.appendChild(form);
			form.submit();
			form.remove();
			vm.$refs.deviceListTable.clearSelection();
		},
		/**
		 * 日志文件更多操作
		 * @parame row:1.查看  2.下载  3.删除 
		*/
		logFileOptClick(row,ev){ 
			var vm = this;  
	    	
	    	vm.rowFileData = row;
	    	
	    	vm.logFileMenus= [
		          {label:'<%=rb.getString("ChaKan")%>',cls:"el-icon el-icon-operation-info CODE_CPE_LOGS hidden" ,code:'view'},
		          {label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" ,code:'download'},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_CPE_LOGS hidden",code:'del'}
		    ]

	    	vm.$nextTick(function(){
	    		document.body.click();
				vm.$refs.logFileMenu.show(ev);
	    	});
	    },

		/**
		 *  日志文件获取点击项数据
		 * @parame ev:点击属性数据
		*/
	    clickLogFileMenues(ev){ //单点击方法 
	    	var vm = this,
				codes = {
					view:this.viewLogFile,
					download:this.downlodFile,
					del:this.delCollectFile
				};
	    	if(codes[ev.code]){
	    		codes[ev.code](vm.rowFileData,'file')
	    	}
	    },
		/**
		 *  查看操作
		 * @param row:当前数据
		*/
	    viewLogFile(row){ 
	    	var vm = this,
				params={
					taskId: row.task_id,
					fileName: row.file_name, 
					fileType: 'CPE'
				};
	    	Object.assign(vm.params_filelist, params);
	    	$("#immediateCollectFileContent").val("");
	    	vm.viewFileDialogVisible = true;
	    },
		/**
		 *  当前页操作
		 * @param row:当前数据
		*/
	    fileSelectChange(row){
			if(row){
				this.viewImmediateLogFileContent(row);
			}
	    },
		/**
		 *  结点击查看上报的日志文件内容信息果
		 * @param rowData:当前数据
		*/
	    viewImmediateLogFileContent(rowData) {
	    	var vm = this,
			    fileType = 'CPE';
	    	$("#immediateCollectFileContent").val("");
	    	
	        if(rowData.file_name == '' || rowData.file_name == null || rowData.un_file_path == '' || rowData.un_file_path == null){
	    		return;
	    	}else{
	    		var params = {
    	   			fileName:rowData.file_name,
    	   			unFilePath:rowData.un_file_path,
    	   			fileType: fileType
    	    	};
	    		$.messager.progress({
		            title : "<%=rb.getString("QingDengDai")%>",
		            text : "<%=rb.getString("JieXiZhong")%>"
		        });
				 
		    	$.post("${ctx}/cell/collect/viewUnZipImmedLogFile.action", params, function (data) {
		    		$.messager.progress("close");
		    		if (data.success) {
		    			if(data.message==""){
		    				showMsg('prompt_msg','<%=rb.getString("WenJianNeiRongWeiKong")%>');
		    			}else{
		    				$("#immediateCollectFileContent").val(data.message);
		    			}
		           } else {
		          		showMsg('prompt_msg','<%=rb.getString("WenJianBuCunZai")%>');
		          	    return;
		           }
		        }, "json");
	    	}
	    },
		//关闭 查看页面
    	cancelDialog(){
    		var vm = this;    		
    		$("#immediateCollectFileContent").val("");
			vm.$refs.fileList.refresh();
			vm.$refs.fileList.setCurrentRow();
	    	vm.viewFileDialogVisible = false;
    	},
		// slide 提交
		submitSlide(){
			var vm = this;
			
			eventBus.$emit('cpe-log-addSubmit');
		},
		// 直接关闭slide事件
        hideSlide(){
			var vm = this;
			vm.$refs.addCpeLogSlide.hide();
			vm.$refs.deviceListTable.refresh();
			vm.$refs.logFileTable.refresh();
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			eventBus.$emit('cpe-log-addCancel');
		},
	},
	mounted(){
		this.init();
		eventBus.$off('hide-cpeLog-slide').$on('hide-cpeLog-slide',this.hideSlide);
	},
})
</script>