<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gsmSettingBackupPage{
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
    overflow: auto;
}
#gsmSettingBackupPage .form-info {
	margin:10px 20px;
}
#gsmSettingBackupPage .form-info .info-title {
	color:#7A7992;
	display:inline-block;
	width:120px;
}
#gsmSettingBackupPage .form-info .el-radio-group .el-radio {
	display:block;
	margin-bottom:20px;
	margin-left:0 !important;
}
#gsmSettingBackupPage .item-title {
	margin:20px 0;
	font-weight:bold;
}
#gsmSettingBackupPage .form-info .el-form-item__label {
	font-size:12px;
	text-align:left;
	line-height:unset;
}
#gsmSettingBackupPage .linkStyle {
	color:#4d84ff;
	font-size:14px;
	text-decoration:underline;
	cursor:pointer;
}
#gsmSettingBackupPage .borderIcon {
	display:inline-block;
	width:22px;
	height:22px;
	line-height:22px;
	text-align:center;
	border:1px solid #d7d7e6;
	border-radius:4px;
	font-size:15px;
	margin-left:5px;
}
</style>

<div id="gsmSettingBackupPage" class="borderPage">
	<el-form ref="form" style="padding:30px 20px;" :model="backupform">
		<div class="form-info" style="margin-bottom:20px;">
			<span class="info-title"><%=rb.getString("BackupRestorePeiZhiWenJian")%></span><span>{{configFile}}</span>
			<span @click="downloadBtn" class="el-icon el-icon-operation-export borderIcon"></span>
		</div>
		<div class="form-info">
			<span @click="importBtn" class="linkStyle" style="margin-left:120px;">Import a configuration file</span>
			<span @click="backupBtn" class="linkStyle" style="margin-left:15px;">Backup from eNB</span>
		</div>
		<div v-if="false" class="form-info">
			<div class="item-title"><%=rb.getString("HuiFuLeiXing") %></div>
			<el-form-item prop="type">
				<el-radio-group v-model="backupform.type">
					<el-radio label="1"><%=rb.getString("HuiFuDaoZuiXinGengXinDeWenJianPeiZhi")%></el-radio>
					<el-radio label="4"><%=rb.getString("HuiFuChuChangPeiZhi")%></el-radio>
				</el-radio-group>
			</el-form-item>
		</div>
		
		<div class="form-info">
			<el-button @click="restoreBtn" type="primary">Restore Now</el-button> <span v-if="false" class="linkStyle" style="margin-left:10px;">View Backip&Restore List</span>
		</div>
		
	</el-form>
	
	<!-- 导入文件框 -->
	<el-dialog title='<%=rb.getString("DaoRu")%>' :visible.sync="showConfirmInfo" width="520" class="backupRestoreImport" append-to-body
		:close-on-click-modal="false"  
		@close="closeImport">
		<el-form :model="importForm" ref="importForm" label-position="left" :rules='importRules'>
			<el-form-item label="<%=rb.getString("BackupRestorePeiZhiWenJian")%>" label-width="160px" prop="fileName">
				<el-upload 
					:before-upload='beforeUpload' 
					:on-success='checkFile' 
					:on-change="fileChange"  
					:show-file-list=false ref="upload"
					:action="importForm.uploadFileUrl"
					:data="fileParams" 
					name="uploadFile" 
					:auto-upload="false" 
					accept=".xml">
					<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:260px;">
						<a slot="suffix" class="el-icon el-icon-operation-import" style="padding-top:4px;" @click="fileSelect"></a>						
					</el-input>	
					<span style="margin-left:10px;">.xml</span>							
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
					
			</el-form-item>  
						
		</el-form>
		
		<div slot="footer">
			<div class="" style="text-align:left;padding:0 20px 10px;">
				<el-button type="primary" @click="fileImport"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeImport"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</div> 	
	</el-dialog>
</div>
<script>
var gsmSettingBackupPageVue = new Vue({
	el: '#gsmSettingBackupPage', 
	data() {
		var vm = this;
		var eNBValidateFileName = function(rule,value,callback) { // 校验设备
        	value = vm.fileName;
			if( value === '' || value === null || value === undefined) {
				callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
			}else if(!fileFormatMatch(value,"xml")){
                callback(new Error("<%=rb.getString("DangQianZhiChixmlGeShiAll")%>"))
            }else {
				callback();
			}
		};
		return {
			backupform:{
				type:'1',
				rawMode:true,
				file:'',
				
			},
			configFile:'',
			backupStatus:'',
			restoreStatus:'',
			latestUpdateTime:'',
			showConfirmInfo:false,
			fileParams:{},              
            fileName:'',	            					
			showFileTip:false,
			fileList:[],
			filePath:'',
			importForm: {
                uploadFileUrl: ''
           	},
           	importRules: {           		
           		fileName:[
                	{required:true, validator: eNBValidateFileName},
                ]           
            },
            showOrHideRestoreStatus: false,
            showOrHideBackupStatus:false,
			code:'',
			sn:''
		};
	},
	computed: {
		isCloud() {
			return isCloud == 'true';
		},
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		
	},
	watch:{
		
	},
	methods: {
		//备份与恢复初始化请求数据
	    init(row){
			var vm = this;
			vm.code = row.small_cell_code;
			vm.sn = row.serial_number;
			
			axios.post('${ctx}/task/enb/config/backupRestore/getDeviceFileInfo.action',stringify({
    			timeZone : timeZone,
    			serial_number : vm.sn,
    			cell_code : vm.code
	    	})).then(function(response){
	    		var data = response.data;
	    		if(Object.keys(data).length != 0){
					vm.configFile = data.file_name;										
					vm.latestUpdateTime = data.update_time;
					
					//如果未返回备份状态字段
					if(data.backup_status === '' || data.backup_status  === null || data.backup_status === undefined){
						vm.showOrHideBackupStatus = false;
					}else{
						vm.backupStatus = data.backup_status;
						vm.showOrHideBackupStatus = true;
					}
					//如果未返回恢复状态字段
					if(data.restore_status === '' || data.restore_status  === null || data.restore_status === undefined){
						vm.showOrHideRestoreStatus = false;
					}else{
						vm.restoreStatus = data.restore_status;						
						vm.showOrHideRestoreStatus = true;
					}
				}
	    	}).catch(function(error){
	    		
	    	})
		},
		//导入文件
		importBtn(){
			this.showConfirmInfo = true;
		},
		/**
		* 文件上传成功函数 
		* @param res{object}   返回信息
		* @param file{object}  文件信息
		*/
		checkFile(res,file){    //发送请求，校验device文件内容 
			var vm = this;
			if(res.success){
				if(res.suc_count>0){
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				}else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
				vm.showConfirmInfo = false;
				vm.closeFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.upload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		fileChange(file,fileList){ 
			var vm = this;
			vm.fileName = file.name;
			vm.fileParams.FileName = file.name;
		},
		// 选择文件
		fileSelect(){  
			var vm =this;
			vm.$refs.upload.clearFiles();
			vm.$refs['file_up'].click();
		},
		// 移除导入文件
		closeFileSelect(){
			var vm = this;
			vm.fileName = '';			
			vm.$refs.upload.clearFiles();
		},
		/**
		* 文件上传之前
		* @param file{object}   文件信息
		*/ 
		beforeUpload(file){
			var vm = this,
				fileName = file.name,
				fileSize = file.size,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
			fd.append('uploadFile',file); //文件流
			fd.append('uploadFileName',fileName);//文件名
			fd.append('fileSize',fileSize);//文件名	大小
			fd.append('serial_number',vm.sn);//基站编码
			axios.post("${ctx}/task/enb/config/backupRestore/single/importFile.action",fd,config).then(function(res){
				if(res.data["success"]){	
					vm.$message.success('<%=rb.getString("ChengGong")%>');						
					vm.showConfirmInfo = false;
					vm.fileList = [];
					vm.fileName = '';
					vm.$refs.importForm.resetFields();	
					vm.init();
				}else{
					vm.$message.error(res.data["message"]);
					vm.showConfirmInfo = false;
				}
			})
			return false;
		},
		/*确定导入*/
        fileImport() {
			var vm = this;
			vm.$refs.importForm.validate((valid) => {
                if (valid) {
                	vm.$refs.upload.submit();                   	
                }
            }) 				
		}, 
		// 关闭导入弹出框
		closeImport(){
			var vm = this;
			vm.showConfirmInfo = false;	
			vm.fileList = [];
			vm.fileName = '';
			vm.$refs.importForm.resetFields();
		},
		// 下载文件
		downloadBtn(){
			var vm = this;
			if(vm.configFile == ''){
				vm.$message.info('<%=rb.getString("DangQianWuPiPeiWenJian")%>')
			}else{
				exportByForm("${ctx}/task/enb/config/backupRestore/single/exportFile.action",{		        	
					fileName: vm.configFile
		        });
			}
		},
		// 备份任务
		backupBtn(){
			var vm = this;
			vm.$confirm('<%=rb.getString("QueDingBeiFenWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/enb/config/backupRestore/addBackupRestoreTask.action',stringify({
	    			taskType : 'backup',
	    			executeMode : 'active',	    			
	    			startTime : '',
	    			timeZone : timeZone,
	    			cellCodes : vm.code
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.init();
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()		
		},
		//恢复任务
		restoreBtn(){
			var vm = this;
			vm.$confirm('<%=rb.getString("QueDingHuiFuWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/enb/config/backupRestore/addBackupRestoreTask.action',stringify({
	    			taskType : 'restore',
	    			executeMode : 'active',	    			
	    			startTime : '',
	    			timeZone : timeZone,
	    			cellCodes : vm.code
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.init();
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){})
	    	}).catch()	
		},
		// 备份与恢复刷新
		refreshBackupRestoreBtn(){
			this.init();
		},
	},
	mounted() {
		eventBus.$off("gsm-data").$on("gsm-data",this.init)
	}
});

</script>
