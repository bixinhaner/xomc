<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
.borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
}
.form-info {
	margin:10px 20px;
}
.form-info .info-title {
	color:#7A7992;
	display:inline-block;
	width:120px;
}
.form-info .el-radio-group .el-radio {
	display:block;
	margin-bottom:20px;
	margin-left:0 !important;
}
.item-title {
	margin:20px 0;
	font-weight:bold;
}
.form-info .el-form-item__label {
	font-size:12px;
	text-align:left;
	line-height:unset;
}
.linkStyle {
	color:#4d84ff;
	font-size:14px;
	text-decoration:underline;
	cursor:pointer;
}
.borderIcon {
	display:inline-block;
	width:26px;
	height:26px;
	line-height:26px;
	text-align:center;
	border:1px solid #d7d7e6;
	border-radius:4px;
}
.curStatus{
	font-size:18px;
	margin-right:5px;
}
.pro-run{
	background: url('${ctx}/css/images/bi/enb_progress.gif') no-repeat center;
}
#gnbLogsPage .form-info .el-ctable-toolbar{
    padding: 10px 0px!important;
}
</style>

<div id="gnbLogsPage" class="borderPage">
	
	<div class="form-info">
		<el-ctable 
            ref="deviceReportLogTable" 
            :rownumber="true" 
            id="gnbDeviceReportLogTable" 
            :time="6" 
            :url="reportLogTableUrl" 
            :query-params="queryParams_reportLog" 
            height="400px" pagination="true"
            style="border: 1px solid #D5DCEC;"
        >
			<template slot="toolbar">
				<div style="padding:0 20px;">
					<span class="titleClass"> File List</span>
					<div style="float:right;" v-if='false'>
						<el-button @click="downlodFile('all')" icon="el-icon el-icon-operation-download"><%=rb.getString("XiaZai")%></el-button>
						<el-button @click="delCollectFile('all')" icon="el-icon el-icon-operation-delete"><%=rb.getString("ShanChu")%></el-button>
					</div>
				</div>
			</template>
			
			<el-table-column label='' width="65" prop="">
				<template slot-scope="scope">
	            	<div class="el-icon el-icon-operation-download" @click="downlodFile(scope.row,event)" ></div>
	            	<div class="el-icon el-icon-operation-delete CODE_GNB_LOGS hidden" @click="delCollectFile(scope.row,event)"  ></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="150" prop="file_name"></el-table-column>
			<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150" prop="upload_time"></el-table-column>
		</el-ctable>
	</div>
	
	<div class="form-info commonFlex" style='padding-bottom: 50px;' v-if="isWritable">
		<span><%=rb.getString("ZhuangTai")%></span> 
		<div v-html='gnbDeviceLogStatus' style='padding-left: 20px;'></div>
	</div>
	
	<div class="form-info" v-if="isWritable">
		<el-button @click="collectLogs" :disabled="disableLog" type="primary">Collect Log</el-button> 
		<span v-if="false" class="linkStyle" style="margin-left:10px;">View Logs Tasks</span>
	</div>

	<form id="gnbExportEventLog" style="display:none" method="post"></form>
	 
</div>
<script>
var gnbLogsPage = new Vue({
	el: '#gnbLogsPage', 
	data() {
		
		return {
			code:'',
			sn:'',
			status:'',
			reportLogTaskUrl:'${ctx}/cell/collect/getImmediateCollectLogTaskPageList.action',
			queryParams_reportTask:{
				timeZone: timeZone,
				device_code: '',
				device_type: 'eNB',
				page:1,
				rows:50,
				sort:'',
				order:'',
                isGnb: 1
			},
			
			reportLogTableUrl:'',
			queryParams_reportLog:{
				timeZone: timeZone,
				taskId: "",
                isGnb: 1
			},
			rowTaskData:[],
			rowData:[],
		    disableLog:true,
		    gnbDeviceLogStatus: '',
            seelctedRow: ''
		};
	},
	computed: {
		isCloud() {
			return isCloud == 'true';
		},
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		isWritable() {
            return writableMap.CODE_GNB_LOGS == true;
        }
	},
	watch:{
		
	},
	methods: {
		init(row,sasStatus) {
			var vm = this,
                code = row.small_cell_code,
                sn = row.serial_number,
                status = row.connection_status;
            
            vm.seelctedRow = row;
			vm.code = code;
			vm.sn = sn;
			vm.status = status;
			
			vm.disableLog = ['On','updating'].includes(vm.status) ? false : true;
			vm.queryParams_reportTask.device_code = code;
			
			if(window.gnbUpdateRowDataTimer){
				clearInterval(window.gnbUpdateRowDataTimer);
			}
			window.gnbUpdateRowDataTimer = setInterval(function(){    
				var gnbLogPageCtn = $("#gnbLogsPage");			
				if(!gnbLogPageCtn.length) {
					clearInterval(window.gnbUpdateRowDataTimer);
					return;
				}
				
				axios.post(vm.reportLogTaskUrl,stringify(vm.queryParams_reportTask)).then(function(response){
					let data = response.data;
					if(data.rows.length > 0){
						vm.rowTaskData = data.rows[0];
						vm.queryParams_reportLog.taskId = data.rows[0].task_id;
						vm.reportLogTableUrl = '${ctx}/cell/collect/getImmediateCollectLogFileDataList.action';
						
						if(vm.rowTaskData.execute_type == 'Immediately'){
							vm.rowTaskData.task_status == 0 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span><%=rb.getString("DengDai")%>');
							vm.rowTaskData.task_status == 1 && (vm.gnbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
							vm.rowTaskData.task_status == 2 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
							vm.rowTaskData.task_status == 3 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
							vm.rowTaskData.task_status == 4 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
						}else{
							vm.rowTaskData.task_status == 0 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus""></span><%=rb.getString("DengDai")%>');
							vm.rowTaskData.task_status == 1 && (vm.gnbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
							vm.rowTaskData.task_status == 2 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
							vm.rowTaskData.task_status == 3 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
							vm.rowTaskData.task_status == 4 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
							vm.rowTaskData.task_status == 5 && (vm.gnbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("ZhongZhiZhong")%>');
							vm.rowTaskData.task_status == 6 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
							vm.rowTaskData.task_status == 7 && (vm.gnbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
							vm.rowTaskData.task_status == 8 && (vm.gnbDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span<span><%=rb.getString("ZhouQiShangBaoSheZhiChengGongCQSX")%></span>');
						}
					}
					
				}).catch(function(error){})
			},6000);
		},
		// 日志收集事件
		collectLogs(){
			var vm = this;
			var	params = {
					timeZone: timeZone,
					serial_number: vm.sn,
					isReboot: 'false',
					device_code: vm.code,
					device_type: 'eNB',
					execute_type: 'Immediately',
					reportPeriod: '',
					start_time: undefined,
					end_time: undefined,
                    isGnb: 1
				},
				url = '${ctx}/cell/collect/goImmediateCollectLogFile.action',
				message = '<%=rb.getString("ChengGong")%>';
		
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message: message,
						type:'success',
					})
					vm.init(vm.seelctedRow);
				}else if(data.responseCode == "901"){
					vm.$message.error(data.message);
				}else if(data.responseCode == "401"){//存在未完成的任务时，再次创建任务失败，给出提示
					vm.$message.error(data["message"]);
				}else{
					vm.$message.error('<%=rb.getString("ShouJiShiBai")%>');
				}
				vm.disableLog = false;
			})
		},
    	/**
		 *  下载文件
		 * @param row:当前数据
		*/
	    downlodFile(row){ 
	    	var isRowParam = (typeof row == 'object');
	    	var vm = this;
	    	if(!vm.rowTaskData){
	    		showMsg('prompt_msg','<%=rb.getString("MeiYouYaoXiaZaiDeWenJian")%>');
	            return;
	    	}
	    	
	    	var taskId = isRowParam? row.task_id : (vm.$root.rowTaskData.task_id || '');
	    	
	    	var fileName = isRowParam? row.file_name : (vm.$root.rowTaskData.file_name || ''),
	    		params = {},
	    		checkURL = '';
	        
			checkURL = '${ctx}/cell/collect/getDownloadFileNumber.action';   		
			url = "${ctx}/cell/collect/doDownloadImmediateCollectLogFile.action"
			params = {
				taskIds: taskId,
				timeZone: timeZone,
				fileName : fileName
			};

    		axios.post(checkURL,stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(data.length > 0 || data.fileNum > 0){
	    			vm.createForm(url,params);
	    		}else{
	    			vm.$message.error('<%=rb.getString("WenJianBuCunZai")%>')
	    		}
	    	}).catch(function(error){
	    		
	    	})
   
	    },
	    /**
		 * 跳转from表格
		 * @param url:地址
		 * @param params：跳转参数
		*/
		createForm(url,param) {
			var body = document.querySelector('body'),
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
		},
		/**
		 * 删除任务  -- 设备上报日志 、告警日志 
		 * @param row:当前数据
		*/
	    delCollectFile(row,status,rowData){
	    	var vm = this , url='' ,params, 
	    		id = ''
	    		fileName = '';
	    	if(!vm.rowTaskData){
	    		showMsg('prompt_msg','<%=rb.getString("MeiYouWenJian")%>');
	            return;
	    	}
	    	if(typeof row == 'object') {
	    		id = row.task_id
	    		fileName = row.file_name;
	    	}else {
	    		id = (vm.$root.rowTaskData.task_id || '');
	    	}
	    	
			url = '${ctx}/cell/collect/doClearImmediateCollectLogFile.action';
			params = {
                taskIds:id,
                fileName : fileName
            };
			
	    	vm.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)
	    		).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.$refs.ctable.refresh()
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
			    		vm.init(vm.seelctedRow);
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data",this.init)
	}
});

</script>
