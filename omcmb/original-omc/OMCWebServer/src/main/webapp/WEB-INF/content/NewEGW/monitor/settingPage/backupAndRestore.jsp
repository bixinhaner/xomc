<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwBackupAndRestorePage .importantBtn{
    height: 28px;
    line-height: 28px;
    padding: 0px 10px;
    margin-left: 10px;
}
.borderPage {
	border:1px solid #d5dcec;
	border-radius:10px;
	height:100%;
	background:#fff;
}
.form-info {
	margin:10px 20px;
	display: flex;
	
}
.form-info .info-title {
	color:#7A7992;
	display:inline-block;
	width:120px;
}
.form-info .el-radio-group .el-radio {
	display:inline-block;
	margin-right:20px;
	margin-bottom:20px;
	margin-left:0 !important;
}
.item-title {
	margin:20px 0;
	width: 200px;
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
.fileNameCls{
	font-weight: 550;
}
#egwBackupAndRestorePage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	padding:0px 20px;
	height:100%;
    display: flex;
    flex-direction: column;
    position: relative;
	overflow: auto;
}
#egwBackupAndRestorePage .itemMainBoxTitle {
	height:50px;
    line-height: 50px;
	font-size:14px;
	font-weight:bold;
}
#egwBackupAndRestorePage .itemTableBoxCls {
	flex: 1;
	min-height: 200px;
	border-top: 1px  solid #d5dcec;
}
#egwBackupAndRestorePage .tableMainBoxCls{
    height:calc(100% - 60px);
    border:1px solid #d5dcec;
    box-sizing: border-box;
}
.egwSettingAddDialogCls .el-input__suffix{
    height: 26px;
    display: flex;
    align-items: center;
}
#egwBackupAndRestorePage .importEgwConfigFilePromptCls{
	font-size: 12px;
	color: #BBBBBB;
	position: relative;
	top: 0px;
	margin-left: 140px;
}
#egwBackupAndRestorePage .importEgwConfigFilePromptCls .el-icon:before{
	color: #BBBBBB;
	font-size: 12px;
}
</style>

<div id="egwBackupAndRestorePage" class="borderPage">
	<div class="itemMainBoxCls">
		<div>
			<el-form ref="form" style="padding:30px 20px;" :model="backupform">
				<div class="form-info">
					<div class="item-title">File Type</div>
					<el-form-item prop="type">
						<el-radio-group v-model="backupform.type" size="small">
							<el-radio label="1" border>SigGW4G</el-radio>
							<el-radio label="2" border>SigGW5G</el-radio>
							<el-radio label="3" border>SigGW4/5G</el-radio>
							<el-radio label="4" border>SeGW</el-radio>
							<el-radio label="5" border>SigGW4G+SeGW</el-radio>
							<el-radio label="6" border>SigGW5G+SeGW</el-radio>
							<el-radio label="7" border>SigGW4/5G+SeGW</el-radio>
						</el-radio-group>
					</el-form-item>
				</div>
				<div class="form-info" style="margin-bottom:20px;align-items: center;">
					<span class="info-title"><%=rb.getString("BackupRestorePeiZhiWenJian")%></span>
					<span class="fileNameCls" v-show="configFile">{{configFile}}</span>
					<span class="fileNameCls" v-show="!configFile">No file</span>
					<el-button class="importantBtn" @click="downloadBtn" :disabled="!configFile">
						<i class="el-icon el-icon-operation-export" style="position:relative;top:2px;"></i>
							Export file
					</el-button>
				</div>
				<div class="form-info">
					<span @click="importBtn" class="linkStyle" style="margin-left:120px;">Import a configuration file</span>
				</div>
				<div class="importEgwConfigFilePromptCls"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><%=rb.getString("DaoRuEgwPeiZhiWenJianTiShi")%></div>
				<div class="form-info">
					<el-button @click="restoreBtn" type="primary">Restore Now</el-button>
					<el-button @click="backupBtn" type="primary">Backup Now</el-button>
				</div>
			</el-form>
		</div>
		<div class="itemTableBoxCls" >
			 <div class="itemMainBoxTitle"><%=rb.getString("JieGuo")%></div>
			 <div class="tableMainBoxCls">
				<el-ctable 
					ref="backupAndRestoreTable" 
					:rownumber="true" 
					:time="6" 
					id="backupAndRestoreTable" 
					:url="backupAndRestoreUrl"
					@load-success="tableLoadSuccess" 
					:query-params="resultsQueryParams" 
					height="100%" 
					pagination="true"
				 >
                    <el-table-column prop="file_type" label="File Type" min-width="120">
						<template slot-scope="scope">
							<div v-html="fileTypeTableResult(scope.row.file_type)"></div>
						</template>
					</el-table-column>
					<el-table-column prop="task_type" label="<%=rb.getString("Type")%>" min-width="100">
						<template slot-scope="scope">
							<span v-if="scope.row.task_type == '1'"><%=rb.getString("BeiFen")%></span>
							<span v-if="scope.row.task_type == '2'"><%=rb.getString("HuiFu")%></span>
						</template>
					</el-table-column>
					<el-table-column prop="file_name" label="<%=rb.getString("BackupRestorePeiZhiWenJian")%>" min-width="200"></el-table-column>	
					<el-table-column prop="task_status" label="<%=rb.getString("ZhuangTai")%>" min-width="80">
						<template slot-scope="scope">
							<div v-html="resultTableStatus(scope.row.task_status)"></div>
						</template>
					</el-table-column>
					<el-table-column prop="task_result" label="<%=rb.getString("JieGuo")%>" min-width="80"> 
						<template slot-scope="scope">
							<div v-html="resultTableResult(scope.row.task_result)"></div>
						</template>
					</el-table-column>
					<el-table-column prop="failure_reason" label="<%=rb.getString("ShiBaiYuanYin")%>" min-width="150"></el-table-column>
					<el-table-column prop="start_time" label="<%=rb.getString("KaiShiShiJian")%>" min-width="100"></el-table-column>
					<el-table-column prop="end_time" label="<%=rb.getString("JieShuShiJian")%>" min-width="100"></el-table-column>
                </el-ctable>
            </div>
		</div>
	</div>
	<!-- 导入文件框 -->
	<el-dialog title='<%=rb.getString("DaoRu")%>' class="egwSettingAddDialogCls" :visible.sync="showConfirmInfo" width="520" class="backupRestoreImport" append-to-body
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
						<a slot="suffix" class="el-icon el-icon-operation-import" @click="fileSelect"></a>						
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
var egwBackupAndRestorePage = new Vue({
	el: '#egwBackupAndRestorePage', 
	data() {
		var vm = this;
		var egwValidateFileName = function(rule,value,callback) { // 校验设备
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
			egwCode:'',
            egwSn:'',
			backupform:{
				type:'1',
				rawMode:true,
				file:'',
			},
			configFileList:{
				'1':'',
				'2':'',
				'3':'',
				'4':'',
				'5':'',
				'6':'',
				'7':'',
			},
			showConfirmInfo:false,
			fileParams:{},              
            fileName:'',	            					
			fileList:[],
			importForm: {
                uploadFileUrl: ''
           	},
           	importRules: {           		
           		fileName:[
                	{required:true, validator: egwValidateFileName},
                ]           
            },
			backupAndRestoreUrl:'${ctx}/egw/backuprestore/getTaskList.action',
			resultsQueryParams:{
				timeZone:timeZone,
				egwCode:'',
			}
		};
	},
	computed: {
		isCloud() {
			return isCloud == 'true';
		},
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		configFile(){
			var  type = this.backupform.type;
			return this.configFileList[type]
		}
	},
	watch:{
		
	},
	methods: {
		// 初始化
	    init(row,code,sn,status){
			var vm = this;
			vm.egwCode = code;
			vm.egwSn = sn;
			vm.resultsQueryParams.egwCode = vm.egwCode;
			vm.initFile();
		},
		//备份与恢复初始化请求数据
	    initFile(){
			var vm = this;
			
			axios.post('${ctx}/egw/backuprestore/getFileInfo.action',stringify({
    			timeZone : timeZone,
    			egwCode : vm.egwCode
	    	})).then(function(response){
	    		var data = response.data;

				Object.keys(vm.configFileList).map((key)=>{
					vm.configFileList[key] = data[key]
				})						
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
			fd.append('egwCode',vm.egwCode);//基站编码
			fd.append('type',vm.backupform.type);//基站编码
			axios.post("${ctx}/egw/backuprestore/importFile.action",fd,config).then(function(res){
				if(res.data["success"]){	
					vm.$message.success('<%=rb.getString("ChengGong")%>');						
					vm.showConfirmInfo = false;
					vm.fileList = [];
					vm.fileName = '';
					vm.$refs.importForm.resetFields();	
					vm.initFile();
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
				exportByForm("${ctx}/egw/backuprestore/exportFile.action",{	
					egwCode:vm.egwCode,	        	
					type: vm.backupform.type,
					fileName:vm.configFile
		        });
			}
		},
		// 备份任务
		backupBtn(){
			var vm = this,
				params = {
					taskType : '1',
					fileType:vm.backupform.type,
	    			timeZone : timeZone,
	    			egwCode : vm.egwCode
				};
			vm.$confirm('<%=rb.getString("QueDingBeiFenWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/egw/backuprestore/addTask.action',stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.initFile();
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
			var vm = this,
				params = {
					taskType : '2',
					fileType:vm.backupform.type,
	    			timeZone : timeZone,
	    			egwCode : vm.egwCode
				};
			vm.$confirm('<%=rb.getString("QueDingHuiFuWenJian")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/egw/backuprestore/addTask.action',stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.initFile();
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
			this.initFile();
		},
		resultTableStatus(cellValue){
			var statusObj = {
				'1':'<%=rb.getString("DengDai")%>',	
				'2':'<%=rb.getString("JinXingZhong")%>',	
				'3':'<%=rb.getString("ZanTing")%>',	
				'4':'<%=rb.getString("YiJieShu")%>',	
				'5':'<%=rb.getString("JinXingZhong")%>',
				'6':'<%=rb.getString("ZhongJianBanBenShengJiZhong")%>',
				'7':'<%=rb.getString("MoKuaiBanBenShengJiZhong")%>',
				'8':'<%=rb.getString("DiBanBanBenShengJiZhong")%>',
				'9':'<%=rb.getString("CanShuPeiZhiZhong")%>',
				'':'',	
			}
			return statusObj[cellValue];
		},
		resultTableResult(cellValue){
			var resultObj = {
				'1' : '<%=rb.getString("ChengGong")%>',
				'3' : '<%=rb.getString("ShiBai")%>',
				'' : '',
			}
			return resultObj[cellValue];
		},
		fileTypeTableResult(cellValue){
			var codes = {
				'1':'SigGW4G',	
				'2':'SigGW5G',	
				'3':'SigGW4/5G',	
				'4':'SeGW',	
				'5':'SigGW4G+SeGW',
				'6':'SigGW5G+SeGW',
				'7':'SigGW4/5G+SeGW',
				'' : '',
			}
			return codes[cellValue];
		},
		// 备份恢复任务数据成功回调
		tableLoadSuccess(){
			var vm = this;
			vm.refreshBackupRestoreBtn();
		},
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
