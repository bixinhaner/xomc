<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
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
#gnbBackupAndRestorePage .exportFileBtn {
    height: 28px;
    line-height: 28px;
    padding: 0 10px;
    margin-left: 10px;
    border-radius: 4px;
    border: 1px solid #DFE2EE;
    background: #F6F7FB;
}
#gnbBackupAndRestorePage .mainBoxCls{
	padding: 30px 40px;
	height: 100%;
    position: relative;
}
#gnbBackupAndRestorePage .downloadBtnTips {
    display: block;
    margin: 70px 0 0 0;
}
#gnbBackupAndRestorePage .el-form-item__label {
    line-height: 28px;
}
.gnbSettingAddDialog .el-input__suffix {
    top: 3px;
}
.gnbSettingAddDialog .el-dialog__body {
    padding: 20px 20px 40px;
}
.gnbSettingAddDialog .el-dialog__footer {
    border-top: 1px solid #D5DCEC;
    height: 48px;
    line-height: 48px;
    padding: 0;
}
</style>

<div id="gnbBackupAndRestorePage" class="commonWarp" style='background: #FFFFFF;'>
	<div class="mainBoxCls">
		<el-form ref="form" :model="gnbBackupRestoreForm">
            <div class="form-info" style="margin-bottom:20px;align-items: center;">
                <span class="commonTextNormal14" style='width: 130px; display: inline-block;'><%=rb.getString("BackupRestorePeiZhiWenJian")%></span>
                <span class="commonSize14" v-show="gnbConfigFile">{{gnbConfigFile}}</span>
                <span class="commonSize14" v-show="!gnbConfigFile">No file</span>
                <el-button class="commonNotes12 exportFileBtn" @click="gnbDownloadBtn" :disabled="!gnbConfigFile">
                    <i class="el-icon el-icon-operation-export" style="font-size: 16px;"></i>
                    Export file
                </el-button>
            </div>
            <span @click="gnbImportBtn" class="commonImportSize14" style="margin-left:130px; text-decoration:underline; cursor:pointer;">Import a configuration file</span>
            
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

			<div class='commonFlex'>
                <el-button class='downloadBtnTips' @click="gnbRestoreBtn" type="primary">Restore Now</el-button>
                <el-button class='downloadBtnTips' @click="gnbBackupBtn" type="primary" style='margin-left: 10px;'>Backup Now</el-button>
            </div>
        </el-form>
	</div>
	<!-- 导入文件框 -->
    <el-dialog title='<%=rb.getString("DaoRu")%>' class="gnbSettingAddDialog" :visible.sync="gnbShowConfirmInfo" width="420" class="backupRestoreImport" append-to-body
        :close-on-click-modal="false"
        @close="gnbCloseImport">
        <el-form :model="gnbImportForm" ref="gnbImportForm" label-position="top" :rules='gnbImportRules'>
            <el-form-item prop="fileName">
                <div slot="label" class='commonFlex'>
                    <p class='commonSize14'><%=rb.getString("BackupRestorePeiZhiWenJian")%> </p>
                    <p class='commonNotes12'>(<%=rb.getString("DangQianZhiChixmlGeShiAll")%>)</p>
                </div>
                <el-upload
                    :before-upload='beforeUpload'
                    :on-success='checkFile'
                    :on-change="fileChange"
                    :show-file-list=false ref="upload"
                    :action="gnbImportForm.uploadFileUrl"
                    :data="fileParams"
                    name="uploadFile"
                    :auto-upload="false"
                    accept=".xml">
                    <el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style='width: 320px;'>
                        <a slot="suffix" class="el-icon el-icon-operation-import" @click="fileSelect"></a>
                    </el-input>
                    <a slot="trigger" ref="file_up"></a>
                </el-upload>
            </el-form-item>
        </el-form>
        <div slot="footer">
            <div style="margin-left: 20px;">
                <el-button type="primary" @click="gnbFileImport"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="gnbCloseImport"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </div>
    </el-dialog>
</div>
<script>
var gnbBackupAndRestoreVue = new Vue({
	el: '#gnbBackupAndRestorePage',
	data() {
		var vm = this;
		var validateFilesName = function(rule,value,callback) { // 校验设备
        	value = vm.fileName;
			if( value === '' || value === null || value === undefined) {
				callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
			}else if(!fileFormatMatch(value,"xml")){
                callback(new Error('<%=rb.getString("DangQianZhiChixmlGeShiAll")%>'));
            }else {
				callback();
			}
		};
		return {
			gnbCellCodes: '',
            gnbSn: '',
			gnbBackupRestoreForm: {
				rawMode: true,
				file: '',
			},
			gnbConfigFile: '',
			gnbShowConfirmInfo: false,
			fileParams: {},
            fileName: '',
			fileList: [],
			gnbImportForm: {
                uploadFileUrl: ''
           	},
           	gnbImportRules: {
           		fileName: [
                	{ validator: validateFilesName}
                ]
            },
			
            showOrHideRestoreStatus: false,
            showOrHideBackupStatus:false,
			backupStatus:'',
			restoreStatus:'',
		};
	},
	methods: {
		// 初始化
	    gnbBackRestoreInit(row, sasEnableStatus){
			var vm = this;

			vm.gnbCellCodes = row.small_cell_code;
			vm.gnbSn = row.serial_number;
			vm.gnbInitFile();
		},
		//备份与恢复初始化请求数据
	    gnbInitFile(){
			var vm = this;

			axios.post('${ctx}/task/enb/config/backupRestore/getDeviceFileInfo.action',stringify({
    			timeZone: timeZone,
    			serial_number: vm.gnbSn,
    			cell_code: vm.gnbCellCodes
    			//isGnb: 1
	    	})).then(function(response){
	    		var data = response.data;
                /*var data = {
                    'file_name': '120000000000182_CFG.xml'
                }*/
                if(Object.keys(data).length != 0){
                    if(data.file_name === '' || data.file_name === null || data.file_name === undefined){
                        vm.gnbConfigFile = '';
                    }else {
                        vm.gnbConfigFile = data.file_name;
                    }

					// 状态等回显 ...
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
		gnbImportBtn(){
			this.gnbShowConfirmInfo = true;
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
				vm.gnbShowConfirmInfo = false;
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
			fd.append('serial_number',vm.gnbSn);//基站编码
			//fd.append('isGnb',1);
			axios.post("${ctx}/task/enb/config/backupRestore/single/importFile.action",fd,config).then(function(res){
				if(res.data["success"]){
					vm.$message.success('<%=rb.getString("ChengGong")%>');
					vm.gnbShowConfirmInfo = false;
					vm.fileList = [];
					vm.fileName = '';
					vm.$refs.gnbImportForm.resetFields();
					vm.gnbInitFile();
				}else{
					vm.$message.error(res.data["message"]);
					vm.gnbShowConfirmInfo = false;
				}
			})
			return false;
		},
		/*确定导入*/
        gnbFileImport() {
			var vm = this;

			vm.$refs.gnbImportForm.validate((valid) => {
                if (valid) {
                	vm.$refs.upload.submit();
                }
            })
		},
		// 关闭导入弹出框
		gnbCloseImport(){
			var vm = this;

			vm.gnbShowConfirmInfo = false;
			vm.fileList = [];
			vm.fileName = '';
			vm.$refs.gnbImportForm.resetFields();
		},
		// 下载文件
		gnbDownloadBtn(){
			var vm = this;
			if(vm.gnbConfigFile == ''){
				vm.$message.info('<%=rb.getString("DangQianWuPiPeiWenJian")%>')
			}else{
				exportByForm("${ctx}/task/enb/config/backupRestore/single/exportFile.action",{
					fileName: vm.gnbConfigFile,
					//isGnb: 1
		        });
			}
		},
		// 备份任务
		gnbBackupBtn(){
			var vm = this;
			vm.$confirm('<%=rb.getString("QueDingBeiFenWenJian")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/enb/config/backupRestore/addBackupRestoreTask.action',stringify({
	    			taskType: 'backup',
	    			executeMode: 'active',
	    			startTime: '',
	    			timeZone: timeZone,
	    			cellCodes: vm.gnbCellCodes,
	    			isGnb: 1
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.gnbInitFile();
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
		gnbRestoreBtn(){
			var vm = this;
			vm.$confirm('<%=rb.getString("QueDingHuiFuWenJian")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/enb/config/backupRestore/addBackupRestoreTask.action',stringify({
	    			taskType: 'restore',
	    			executeMode: 'active',
	    			startTime: '',
	    			timeZone: timeZone,
	    			cellCodes: vm.gnbCellCodes,
				    isGnb: 1
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.gnbInitFile();
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
		refreshRestore() {
			var vm = this;
			var ctnDom = document.querySelector('#gnbBackupAndRestorePage');

			if(ctnDom && isVisible(ctnDom)) {
				vm.gnbInitFile();
			}
		}
	},
	mounted() {
		eventBus.$off("gnb-data").$on("gnb-data", this.gnbBackRestoreInit);
		
		if(window.gnbBackupTimer) {
			clearInterval(window.gnbBackupTimer)
		}
		window.gnbBackupTimer = setInterval(this.refreshRestore, 6000);
	}
});

</script>
