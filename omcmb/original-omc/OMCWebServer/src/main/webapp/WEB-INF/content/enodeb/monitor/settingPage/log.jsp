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
.wenJianBox{
	height: 60%;
	display: flex;
	overflow: visible;
	border: 1px solid #d1ecf5;
}
.wenJianBoxPad{
	width: 260px;
	border-right: 1px solid #d1ecf5;
}
.curStatus{
	font-size:18px;
	margin-right:5px;
}
.pro-run{
	background: url('${ctx}/css/images/bi/enb_progress.gif') no-repeat center;
}
.textareaBox .el-textarea {
	width: 100%;height: 100%; border: 0;
}
.textareaBox .el-textarea__inner {border: 0;width: 100%;height: 97%;resize: none; font-size: 12px; color: rgba(0, 0, 0, 0.8) !important;}
#enbLogsPage .el-icon-arrow-right:before{
    content:"\e794" !important;
    color:#C0C4CC !important;
    font-size:12px !important;
    font-weight: 700 !important;
}
</style>

<div id="enbLogsPage" class="borderPage">
	
	<div class="form-info">
		<el-ctable 
            ref="deviceReportLogTable" 
            :rownumber="true" 
            id="deviceReportLogTable" 
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
					<!-- <div class="el-icon el-icon-operation-view" @click="viewLogFile(scope.row,event)"  ></div> -->
	            	<div class="el-icon el-icon-operation-download" @click="downlodFile(scope.row,event)" ></div>
	            	<div class="el-icon el-icon-operation-delete CODE_ENB_LOGS hidden" @click="delCollectFile(scope.row,event)"  ></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="150" prop="file_name"></el-table-column>
			<el-table-column label='<%=rb.getString("GengXinShiJian")%>' min-width="150" prop="upload_time"></el-table-column>
		</el-ctable>
	</div>
	
	<div class="form-info commonFlex" style='padding-bottom: 50px;' v-if="isWritable">
		<span><%=rb.getString("ZhuangTai")%></span> 
		<div v-html='enbDeviceLogStatus' style='padding-left: 20px;'></div>
	</div>
	
	<div class="form-info" v-if="isWritable">
		<el-button @click="collectLogs" :disabled="disableLog" type="primary">Collect Log</el-button> 
		<span v-if="false" class="linkStyle" style="margin-left:10px;">View Logs Tasks</span>
	</div>
		
	<el-dialog :visible.sync="dialogResultVisible" title="<%=rb.getString("WenJianXinXi")%>" append-to-body :close-on-click-modal="false" @close="canceResultlDialog">
	  	<div class="wenJianBox">
			<div class="wenJianBoxPad" style="">
				<el-ctable id="logFileList" ref="fileList" :url="logFileListUrl" :height="'100%'" @row-click="fileSelectChange" :pagination="false"
						:query-params="params_filelist">
					<!-- 主列表 -->
					<el-table-column label='<%=rb.getString("WenJianLieBiao")%>' prop="un_file_name" ></el-table-column>
				</el-ctable>
			</div>
			<div style="width: 100%;padding: 10px;" class='textareaBox'>
			 	<el-input type="textarea" v-model="collectFileContent" class="border border-box" style="padding-left:10px;"></el-input>
			</div>
		</div>
	 </el-dialog>	
	 <form id = "exportEventLog" style="display:none" method="post"></form>
	 
</div>
<script>
var updateRowDataTimer;
var enbLogsPage = new Vue({
	el: '#enbLogsPage', 
	data() {
		
		return {
            enbSelectedRow:{},
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
				order:''
			},
			
			reportLogTableUrl:'',
			queryParams_reportLog:{
				timeZone: timeZone,
				taskId: "",
			},
			
			dialogResultVisible: false,
			params_filelist: {
				taskId: '',
				fileName: '',
				fileType: ''
			},
			logFileListUrl: '${ctx}/cell/collect/doUnZipImmedLogFile.action',
			rowTaskData:[],
			rowData:[],
		    rowDataFile:[],
		    disableLog:true,
		    enbDeviceLogStatus: '',
		    collectFileContent: ''
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
            return writableMap.CODE_ENB_LOGS == true;
        }
	},
	watch:{
		
	},
	methods: {
		init(row,code,sn,status){
			var vm = this;
			vm.code = code;
			vm.sn = sn;
			vm.status = status;
			vm.enbSelectedRow = row;
			vm.disableLog = ['On','updating'].includes(vm.status) ? false : true;
			vm.queryParams_reportTask.device_code = code;
			vm.getTableListData();
			if(updateRowDataTimer){
				clearInterval(updateRowDataTimer);
			}
			updateRowDataTimer = setInterval(function(){    
				var enbLogPageCtn = $("#enbLogsPage");			
				if(!enbLogPageCtn.length) {
					clearInterval(updateRowDataTimer);
					return;
				}
				vm.getTableListData();
			},6000);
		},
        // 请求表格数据
        getTableListData(){
            var vm = this;
            axios.post(vm.reportLogTaskUrl,stringify(vm.queryParams_reportTask)).then(function(response){
                let data = response.data;
                if(data.rows.length > 0){
                    vm.rowTaskData = data.rows[0];
                    vm.queryParams_reportLog.taskId = data.rows[0].task_id;
                    vm.reportLogTableUrl = '${ctx}/cell/collect/getImmediateCollectLogFileDataList.action';
                    
                    if(vm.rowTaskData.execute_type == 'Immediately'){
                        vm.rowTaskData.task_status == 0 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span><%=rb.getString("DengDai")%>');
                        vm.rowTaskData.task_status == 1 && (vm.enbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
                        vm.rowTaskData.task_status == 2 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
                        vm.rowTaskData.task_status == 3 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
                        vm.rowTaskData.task_status == 4 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
                    }else{
                        vm.rowTaskData.task_status == 0 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus""></span><%=rb.getString("DengDai")%>');
                        vm.rowTaskData.task_status == 1 && (vm.enbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
                        vm.rowTaskData.task_status == 2 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-success greenIcon curStatus"></span><%=rb.getString("ChengGong")%>');
                        vm.rowTaskData.task_status == 3 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-failed redIcon curStatus"></span><%=rb.getString("LogsShiBai")%>');
                        vm.rowTaskData.task_status == 4 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
                        vm.rowTaskData.task_status == 5 && (vm.enbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("ZhongZhiZhong")%>');
                        vm.rowTaskData.task_status == 6 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-terminate curStatus"></span><%=rb.getString("ZhongZhi")%>');
                        vm.rowTaskData.task_status == 7 && (vm.enbDeviceLogStatus = '<span style="display:inline-block;width:60px;height:6px;border-radius:10px;" class="pro-run"></span><%=rb.getString("JinXingZhong")%>');
                        vm.rowTaskData.task_status == 8 && (vm.enbDeviceLogStatus = '<span class="el-icon el-icon-status-waiting1 curStatus"></span<span><%=rb.getString("ZhouQiShangBaoSheZhiChengGongCQSX")%></span>');
                    }
                }
			}).catch(function(error){})
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
                    logType: 'deviceLog',
					start_time: undefined,
					end_time: undefined
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
					vm.getTableListData();
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
		 *  查看操作
		 * @param row:当前数据
		*/
	    viewLogFile(row,ev){ 
	    	var vm = this;
	    	
	    	var params={
	    		"taskId": row.task_id,
	    		"fileName": row.file_name, 
	    		fileType: 'enb'
	    	};
	    	Object.assign(vm.params_filelist, params);
	    	vm.collectFileContent = '';
	    	vm.dialogResultVisible = true;
	    	vm.$root.rowData = row;
	    },
	    fileSelectChange(row){
			if(row){
				this.viewImmediateLogFileContent(row);
			}
	    },
		viewImmediateLogFileContent(rowData) {
	    	var vm = this;
	    	vm.collectFileContent = '';
	        if(rowData.file_name == '' || rowData.file_name == null || rowData.un_file_path == '' || rowData.un_file_path == null){
	    		return;
	    	}else{
	    		var params = {
    	   			fileName:rowData.file_name,
    	   			unFilePath:rowData.un_file_path,
    	   			fileType: 'enb'
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
		    				vm.collectFileContent = data.message;
		    			}
		           } else {
		          		showMsg('prompt_msg','<%=rb.getString("WenJianBuCunZai")%>');
		          	    return;
		           }
		        }, "json");
	    	}
	    },
	  //关闭结果->查看页面
    	canceResultlDialog(){
    		var vm = this;    		
    		vm.collectFileContent = '';
			vm.$refs.fileList.refresh();
			vm.$refs.fileList.setCurrentRow();
	    	vm.dialogResultVisible = false;
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
			    		vm.getTableListData();
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
	},
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.init)
	}
});

</script>
