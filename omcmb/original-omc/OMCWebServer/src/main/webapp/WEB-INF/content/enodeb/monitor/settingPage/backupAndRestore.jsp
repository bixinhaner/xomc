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
	width:22px;
	height:22px;
	line-height:22px;
	text-align:center;
	border:1px solid #d7d7e6;
	border-radius:4px;
	font-size:15px;
	margin-left:5px;
}
#enbBackupPage .downloadBtnTips {
	background: rgba(var(--main-color-rgba1),0.1);
	color: var(--main-color);
}
#enbBackupPage .exportFileBtn {
    height: 28px;
    line-height: 28px;
    padding: 0 10px;
    margin-left: 10px;
    border-radius: 4px;
    border: 1px solid #DFE2EE;
    background: #F6F7FB;
}
#enbBackupPage .el-icon-operation-import {
	margin-top: 3px;
}
</style>

<div id="enbBackupPage" class="borderPage" style="position: relative;">
	<div class="el-icon-common-refresh el-icon" @click='refreshBackupRestoreBtn' style="position:absolute;right:15px;top: 15px;font-size:20px;"></div>
	<el-form ref="form" style="padding:30px 20px;" :model="backupform">
		<div class="form-info" style="margin-bottom:20px;">
			<span class="commonTextNormal14" style='width: 150px; display: inline-block;'><%=rb.getString("BackupRestorePeiZhiWenJian")%></span>
			<span class="commonSize14" v-show="configFile">{{configFile}}</span>
			<span class="commonSize14" v-show="!configFile">No file</span>
			<el-button class="commonNotes12 exportFileBtn" @click="downloadBtn" :disabled="!configFile">
				<i class="el-icon el-icon-operation-export" style="font-size: 16px;"></i>
				Export file
			</el-button>
		</div>
		<div class="form-info">
			<span @click="importBtn" class="linkStyle" style="margin-left:150px;">Import a configuration file</span>
		</div>
		<div class="form-info commonFlex" style="margin-bottom:20px; margin-top: 30px;">
			<span class="commonTextNormal14" style='width: 150px; display: inline-block;'><%=rb.getString("BeiFenZhuangTai")%></span>
			<div v-show ="showOrHideBackupStatus">
				<div v-if="backupStatus == 'End'">
					<span class='el-icon el-icon-status-terminate' style='margin-right: 5px;'></span>
					<span><%=rb.getString("YiJieShu")%></span>
				</div>
				<div v-else-if="backupStatus == 'inProgress'">									
					<span class="status_inProgress"></span>
					<span><%=rb.getString("JinXingZhong")%></span>
				</div>
				<div v-else></div>
			</div>
		</div>
		<div class="form-info commonFlex" style="margin-bottom:20px;">
			<span class="commonTextNormal14" style='width: 150px; display: inline-block;'><%=rb.getString("HuiFuZhuangTai")%></span>
			<div v-show ="showOrHideRestoreStatus">
				<div v-if="restoreStatus == 'End'">
					<span class='el-icon el-icon-status-terminate' style='margin-right: 5px;'></span>
					<span><%=rb.getString("YiJieShu")%></span>
				</div>
				<div v-else-if="restoreStatus == 'inProgress'">
					<span class="status_inProgress"></span>
					<span><%=rb.getString("JinXingZhong")%></span>
				</div>
				<div v-else></div> 
			</div>
		</div>
		<div class="form-info" style="margin-top: 70px;">
			<el-button class='downloadBtnTips' @click="restoreBtn" type="primary"><%=rb.getString("LiJiHuiFu")%></el-button>
			<el-button class='downloadBtnTips' @click="backupBtn" type="primary"><%=rb.getString("LiJiBeiFen")%></el-button> <!--NewLiJiBeiFen-->
		</div>
		
	</el-form>
	
	<!-- 导入文件框 -->
	<el-dialog title='<%=rb.getString("DaoRu")%>' :visible.sync="showConfirmInfo" width="550" class="backupRestoreImport" append-to-body
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
					accept=".xml, .nv">
					<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:260px;">
						<a slot="suffix" class="el-icon el-icon-operation-import" @click="fileSelect"></a>						
					</el-input>	
					<span style="margin-left:10px;">.xml, .nv</span>							
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
var enbBackupPage = new Vue({
	el: '#enbBackupPage', 
	data() {
		var vm = this;
		var eNBValidateFileName = function(rule,value,callback) { // 校验设备
        	var value = vm.fileName,
				productType = vm.enbSelectedRow.platformType,
				productFlag = vm.enbSelectedRow.product;
				
			//需要根据产品类型进行判断支持的文件类型
			if( value === '' || value === null || value === undefined) {
				callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
			}else {
				if(productType == 'MLQ' || productFlag == 'MLN'){
					if(!fileFormatMatch(value,"nv")){
						callback(new Error("<%=rb.getString("BeiFeiHuiFuDaoRuWenJianLeiXingNV")%>"))
					}else {
						callback();
					}
				}else{
					if(!fileFormatMatch(value,"xml")){
						callback(new Error("<%=rb.getString("DangQianZhiChixmlGeShiAll")%>"))
					}else {
						callback();
					}
				}
			}
		};
		return {
            enbSelectedRow:{},
			backupform:{
				type:'1',
				rawMode:true,
				file:''
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
			code: '',
			sn: ''
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
	    initFile(row,code,sn,status){
			var vm = this;
            vm.enbSelectedRow = row;
			vm.code = code;
			vm.sn = sn;
            vm.getDeviceFileInfo();
		},
		getDeviceFileInfo(){ 
            var vm = this,
                params = {
                    timeZone : timeZone,
                    serial_number : vm.enbSelectedRow.serial_number,
                    cell_code : vm.enbSelectedRow.small_cell_code
                };
            axios.post('${ctx}/task/enb/config/backupRestore/getDeviceFileInfo.action',stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(Object.keys(data).length != 0){
														
					//如果未返回备份状态字段
					if(data.file_name === '' || data.file_name  === null || data.file_name === undefined){
						vm.configFile = '';
					}else{
						vm.configFile = data.file_name;	
					}
					if(data.update_time === '' || data.update_time  === null || data.update_time === undefined){
						vm.latestUpdateTime = '';
					}else{
						vm.latestUpdateTime = data.update_time;
					}
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
	    	}).catch(function(error){})
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
					vm.getDeviceFileInfo();
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

			if(vm.configFile === '' || vm.configFile  === null || vm.configFile === undefined){
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
		    			vm.getDeviceFileInfo();
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
		    			vm.getDeviceFileInfo();
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
			var vm = this;
			var ctnDom = document.querySelector('#enbBackupPage');

			if(ctnDom && isVisible(ctnDom)) {
				vm.getDeviceFileInfo();
			}
		},
	},
	mounted() {
		eventBus.$off("enb-data").$on("enb-data",this.initFile)

		if(window.enbBackupTimer) {
			clearInterval(window.enbBackupTimer)
		}
		window.enbBackupTimer = setInterval(this.refreshBackupRestoreBtn, 6000);
	}
});

</script>
