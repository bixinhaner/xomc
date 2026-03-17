<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#moduleImportfile .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#moduleImportfile .el-form-item__error{
	white-space: nowrap;
}
#moduleImportfile .el-form-item{
	margin-bottom:30px;
}
#moduleImportfile .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#moduleImportfile .uploadInput .el-input__suffix{
	top:5px !important;
}
.item-flex .el-form-item{
	display: inline-block;
	width: 500px;
}
.tableBox{
	position: relative;
	width: 60%;
}
.tableBox .pairgrid-right{
	top:40px!important;
	height: calc(100% - 40px)!important;
}
.tableBox .el-pairgrid-title{
	top:15px!important;
	right: 15px!important;
}
.tableBox .tableTitleCls{
	margin-bottom: 10px;
}
.tableTitleCls span:nth-child(1){
	color: #000;
	font-size: 14px;
}
.tableTitleCls span:nth-child(2){
	color: #999999;
	font-size: 12px;
	margin-left: 10px;
}
.addModuleNameBox{
	position: absolute;
	right: 0px;
	top: -10px;
}
.filePrompt{
	font-size: 12px;
	color: #BBBBBB;
	position: absolute;
	white-space: nowrap;
	top: 0px;
	left: 310px;
}
.filePrompt .el-icon:before{
	color: #BBBBBB;
	font-size: 12px;
}
</style>

<div id='moduleImportfile'>
	<el-form ref="cpeFileImportForm" :model="importForm" label-width="120px" label-position="left" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<div class="item-flex">
			<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
				<el-upload 
					v-show="operationType == 'add'" 
					:before-upload='beforeUpload' 
					:on-success='checkFile' 
					:on-error='checkFileError' 
					:on-change="fileChange"  
					:show-file-list=false 
					ref="upload"
					:file-list="fileList"
					:action="uploadFileUrl" 
					:data="fileParams" 
					name="uploadFile" 
					:auto-upload="false">
					<el-input style="width:300px;" class="uploadInput" :readonly="true" :value='importForm.fileName' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>								
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
				<el-input v-show="operationType != 'add'" v-model='importForm.fileName' :disabled="viewFlag || editFlag" style="width:300px;"></el-input>
				<div class="filePrompt"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><%=rb.getString("TGZWenJianTiShi")%></div>
			</el-form-item>
			<el-form-item label="<%=rb.getString("BanBen")%>" prop="version" style="margin-left:200px;">
				<el-input v-model='importForm.version' :disabled="viewFlag || editFlag " style="width:200px;"></el-input>
			</el-form-item>
		</div>
		<div class="tableBox">
			<div class="tableTitleCls">
				<span><%=rb.getString("MoKuaiMingCheng")%></span>
			</div>
			<div class="addModuleNameBox" v-if="showPairgrid">
				<span class="el-icon el-icon-circle-add" @click="addModuleNameClick"></span>
			</div>
			<el-form-item label='' label-width="0px" prop='moduleName' style="position: relative;">
				<el-pairgrid v-if="showPairgrid" id="moduleNamePairgrid" @checked-change='moduleNameSelectChange'   ref="moduleNamePairgrid" :rownumber="true" :right-url="moduleNameRightUrl" :left-url="moduleNameLeftUrl" :height="height" :readonly='viewFlag' row-key="module_name" :query-params="queryModuleNameForm" :title="moduleNameTitle" :messages="{placeholder:'<%=rb.getString("MoKuaiMingCheng")%>'}">
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column prop='module_name' label='<%=rb.getString("MoKuaiMingCheng")%>'></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryModuleNameList" placeholder="<%=rb.getString("MoKuaiMingCheng")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='module_name' label='<%=rb.getString("MoKuaiMingCheng")%>'></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable v-else id="'selected_moduleName_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="moduleNameRightUrl" :height="height" front-pagination="true" pagination="true">
					<el-table-column prop='module_name' label='<%=rb.getString("MoKuaiMingCheng")%>'></el-table-column>
				</el-ctable>
			</el-form-item>
			<div class="tableTitleCls">
				<span><%=rb.getString("MuBiaoBanBen")%></span>
				<span></span>
			</div>
			<el-form-item label='' label-width="0px" prop='destVersion' style="position: relative;">
				<el-pairgrid v-if="showPairgrid" @checked-change='destVersionSelectChange' :limit="9"   ref="destVersionPairgrid" :rownumber="true" :right-url="destVersionRightUrl" :left-url="destVersionLeftUrl" :height="height" :readonly='viewFlag' row-key="id" :query-params="queryDestVersionForm" :title="destVersionTitle" :messages="{placeholder:'<%=rb.getString("MuBiaoBanBen")%>'}">
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column prop='version' label='<%=rb.getString("MuBiaoBanBen")%>'></el-table-column>
						<el-table-column prop='model_name' label='<%=rb.getString("ChanPinXingHao")%>'></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDestVersionList" placeholder="<%=rb.getString("MuBiaoBanBen")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='version' label='<%=rb.getString("MuBiaoBanBen")%>'></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable v-else id="'selected_destVersion_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="destVersionRightUrl" :height="height" front-pagination="true" pagination="true">
					<el-table-column prop='version' label='<%=rb.getString("MuBiaoBanBen")%>'></el-table-column>
				</el-ctable>
			</el-form-item>
		</div>
		
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='description'>
			<el-input :disabled="viewFlag" v-model='importForm.description' type='textarea' :rows='3' style='width:600px;'></el-input>
		</el-form-item>
	</el-form>
	<el-dialog title='<%=rb.getString("TianJia")%>' id="addModuleNameDialog" :visible.sync="showAddModuleNameDialog" top="30vh" ref="addModuleNameDialog" :width="addModuleNameDialogWidth" 
			:close-on-click-modal="false"  @close='closeAddModuleName' append-to-body>
			<el-form  
				:model="addModuleNameForm"
				ref="addModuleNameFormBox" 
				:rules="addModuleNameRules" 
				label-position="left" 
				id="addModuleNameForm"
				@submit.native.prevent
			>
				<el-form-item  style="margin-left:40px;margin-top:10px;" label="<%=rb.getString("MoKuaiMingCheng")%>" prop='addModuleName'  label-width="120px">
					<el-input v-model='addModuleNameForm.addModuleName'  style="width:200px;padding-top:7px;"></el-input>
				</el-form-item>
			</el-form>
			<span slot="footer">
				<div>
					<el-button type="primary" @click="addModuleNameSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeAddModuleName"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</span>
	</el-dialog>
</div>
<script>
	new Vue({
		el:'#moduleImportfile',
		data(){
			var vm = this;
			var versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						if(value == vm.defaultVersion){
							callback();
						}else{
							if(vm.importForm.moduleName == ''){
								callback();
							}else{
								axios.get("${ctx}/cell/version/verifyModuleUVExist.action",{
									params:{
										moduleName : vm.importForm.moduleName,
										version : value
									}
								}).then(function(response){
									var data = response.data;
									if(!data["success"]){
										callback(data["message"]);
									}else{
										callback();
									}
								})
							}
							
						}
					}
				}else {
					callback('<%=rb.getString("QingShuRuWenJianBanBen")%>');
				}
			};
			var fileNameValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
					}else if(!fileFormatMatch(value,"tgz")){
						callback('<%=rb.getString("ZhIZhiChiTGZWenJian")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
			var addModuleNameValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("MingChengChangDu")%>1-100');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("MingChengChangDu")%>1-100');
				}
			};
			var validatorDestVersion = function(rule,value,callback) {
				if(value) {
					callback();
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
			var validatorModuleName = function(rule,value,callback) {
				if(value) {
					callback();
				}else {
					callback('<%=rb.getString("QingXuanZeMoKuaiXingHao")%>');
				}
			};
			return{
				importForm:{
					product:'CPE_VERSION',
					fileName:'',
					version:'',
					moduleName:'',
					destVersion:'',
					description:'',
					destVersionId:''
				},
				addModuleNameForm:{
					addModuleName:''
				},
				queryModuleNameForm:{
					searchText:'',
				},
				queryDestVersionForm:{
					searchText:'',
					timeZone:timeZone
				},
				height:'370px',
				moduleNameLeftUrl:'${ctx}/cell/version/queryModuleNames.action',
				moduleNameRightUrl:'',
				moduleNameTitle:['','<%=rb.getString("YiXuanLieBiao")%>'],
				destVersionLeftUrl:'${ctx}/cell/version/queryfileInfosList.action?file_type=9',
				destVersionRightUrl:'',
				destVersionTitle:['','<%=rb.getString("YiXuanLieBiao")%>'],
				importRules:{
					version:[
						{validator: versionValidate}
					],
					fileName:[
						{validator: fileNameValidate}
					],
					moduleName:[
						{validator:validatorModuleName,trigger:'change'}
					],
					// destVersion:[
					// 	{validator:validatorDestVersion,trigger:'change'}
					// ]
				},
				addModuleNameRules:{
					addModuleName:[
						{validator: addModuleNameValidate}
					],
				},
				showPairgrid:true,
				viewFlag : false,
				editFlag:false,
				fileParams:{},            //上传文件时自定义的参数  
				fileList:[],
				uploadFileUrl:'',
				moduleNameList:[],
				operationType:'add',
				fileVersion:'',
				showAddModuleNameDialog:false,
				addModuleNameDialogWidth:'460px',
				fileErrorData:'',
				defaultVersion:'',
                multipartMaxFileSize: multipartMaxFileSize
			}
		},
		computed:{
			
			moduleName(){
				return this.moduleNameList.join(',')
			}
    	},
		methods:{
			// 初始化
			init(type,version){
				var vm = this;
				vm.operationType = type;
				vm.fileVersion = version;
				if(vm.operationType != 'add'){
					vm.getImportInfo()
					if(vm.operationType == 'view'){
						vm.viewFlag = true;
						vm.showPairgrid = false;
					}else{
						vm.editFlag = true;
					}
				}
			},
			// 获取详情信息
			getImportInfo(){
				var vm = this,
					params={
						moduleVersion:vm.fileVersion
					};
				axios.post('${ctx}/cell/version/queryModuleFileInfo.action',stringify(params)).then(function(response){
						var data = response.data;
						['fileName','version','description','moduleName','destVersion'].map((item)=>{
							vm.importForm[item] = data[item];
							vm.defaultVersion = data.version;
						})
						vm.moduleNameRightUrl = '${ctx}/cell/version/querySelectedModuleNameInfo.action?moduleVersion='+vm.fileVersion;
						vm.destVersionRightUrl = '${ctx}/cell/version/querySelectedTargetVersionInfo.action?moduleVersion='+vm.fileVersion;
						initForm(vm.$refs.cpeFileImportForm);
					}).catch(function(error){})
			},
			// 添加模块名称
			addModuleNameClick(){
				var vm = this;
				vm.showAddModuleNameDialog = true;
				vm.addModuleNameForm.addModuleName = '';
				vm.$nextTick(function(){
					vm.$refs.addModuleNameFormBox.clearValidate();
				});
			},
			// 新增模块名称 提交
			addModuleNameSubmit(){
				var vm = this;
				vm.$refs.addModuleNameFormBox.validate((valid) => {
					if(valid){
						vm.$refs.moduleNamePairgrid.appendCheckedRow({module_name:vm.addModuleNameForm.addModuleName});
						vm.showAddModuleNameDialog = false;
					}else{
						return false;
					}
				})
			},
			// 新增模块名称 取消
			closeAddModuleName(){
				var vm = this;
				vm.showAddModuleNameDialog = false;
			},
			// 模块名称列表选择事件
			moduleNameSelectChange(){
				var vm =this,
					data = vm.$refs.moduleNamePairgrid.getData();
					selectedModuleName = '',
					selectedModuleNameList=[];
				if(data.length != 0){
					data.map(function(item){
						selectedModuleNameList.push(item.module_name)
					})
				}
				selectedModuleName = selectedModuleNameList.join(',');
				vm.importForm.moduleName = selectedModuleName;
			},
			// 模块名称列表 模糊搜索
			queryModuleNameList(val){
				var vm = this;
				vm.queryModuleNameForm.searchText = val;
			},
			// 目标版本列表选择事件
			destVersionSelectChange(){
				var vm =this,
					data = vm.$refs.destVersionPairgrid.getData();
					selectedDestVersion = '',
					selectedDestId = '',
					selectedDestVersionList=[],
					selectedDestIdList = [];
				if(data.length != 0){
					data.map(function(item){
						selectedDestVersionList.push(item.version);
						selectedDestIdList.push(item.id)
					})
				}
				selectedDestVersion = selectedDestVersionList.join(',');
				selectedDestId = selectedDestIdList.join(',');
				vm.importForm.destVersion = selectedDestVersion;
				vm.importForm.destVersionId = selectedDestId;
			},
			//目标版本列表 模糊搜索
			queryDestVersionList(val){
				var vm = this;
				vm.queryDestVersionForm.searchText = val;
			},
			/**
			* 文件上传成功函数 
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			*/
			checkFile(res,file){    //发送请求，校验device文件内容 
				var vm = this;
			},
			checkFileError(res,file){
				var vm = this;
			},
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file,fileList){ 
                var vm = this,
				    fileSize = file.size;

                if(fileSize <= vm.multipartMaxFileSize){
                    vm.importForm.fileName = file.name;
                    vm.importForm.version = file.name.substring(0,file.name.lastIndexOf("."));
                    vm.fileParams.FileName = file.name;
                }else{
                    var maxFileSizeMB = Number(vm.multipartMaxFileSize / 1024 / 1024).toFixed(0),
                        fileSizeMB = Number(fileSize / 1024 / 1024).toFixed(0),
                        messageStr = '<%=rb.getString("CollectLogSizeExceedOne")%>' + fileSizeMB + '<%=rb.getString("CollectLogSizeExceedTwo")%>' + maxFileSizeMB + '<%=rb.getString("CollectLogSizeExceedThree")%>';
                    vm.$message({
                        type: 'error',
                        message: messageStr
                    });
                    vm.$refs.upload.clearFiles();
                }
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
				vm.importForm.fileName = '';
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this;
				var fileName = file.name,fileSize = file.size,fileType = '';
				var fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' },
						onUploadProgress:(ev)=>{
							if(ev.lengthComputable || ev.event.lengthComputable) {
								var total = ev.total,
									loaded = ev.loaded,
									percent = 100*loaded/total;
								$('#progressUploadFile').progressbar('setValue', percent.toFixed(2));
							}
						}
					};
				
				fd.append('uploadFile',file); //文件流
				fd.append('newFileName',fileName);//文件名
				fd.append('fileSize',fileSize);//文件大小
				fd.append('desc',vm.importForm.description);//描述
				fd.append('moduleName',vm.importForm.moduleName);
				fd.append('version',vm.importForm.version);
				fd.append('destVersion',vm.importForm.destVersion);
				fd.append('destVersionId',vm.importForm.destVersionId);
				vm.fileErrorData = vm.$refs.upload.uploadFiles[0];
				$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
				$("#winUploadPro").window("open");// 打开进度条窗口
                cpeUpgrade.slideSubmitLoading = true;
				axios.post("${ctx}/cell/version/uploadModuleVersionFile.action",fd,config).then(function(response){
					var data = response.data
					$("#winUploadPro").window("close");// 关闭进度条窗口
					if(data["MD5"]){
						vm.fileErrorData = '';
						$.messager.alert(TiShi,'<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["MD5"]);
						eventBus.$emit('hide-cpeUpgrade-slide');
					}else{
						vm.$message.error(data["message"]);
                        cpeUpgrade.slideSubmitLoading = false;
					}
				})
				return false;
			},
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.cpeFileImportForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-cpeUpgrade-slide');
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-cpeUpgrade-slide');
				}
			},
			submit(){
				var vm = this;
                // 防止多次提交
                if(cpeUpgrade.slideSubmitLoading)return

				vm.$refs.cpeFileImportForm.validate((valid) => {
					if(valid){
						if(vm.operationType == 'add'){
							
							if(vm.fileErrorData){
								vm.$refs.upload.uploadFiles.push(vm.fileErrorData);
							}
							vm.$refs.upload.submit();
						}else{
							vm.fileImportEditSubmit()
						}
					}
				})
			},
			fileImportEditSubmit(){
				var vm = this,
					params={
						moduleVersion:vm.importForm.version,
						moduleName:vm.importForm.moduleName,
						destVersion:vm.importForm.destVersion,
						destVersionId:vm.importForm.destVersionId,
						desc:vm.importForm.description
					};
                cpeUpgrade.slideSubmitLoading = true;
				axios.post('${ctx}/cell/version/updateModuleVersionInfo.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							eventBus.$emit('hide-cpeUpgrade-slide');
						}else{
							vm.$message.error(data["message"]);
                            cpeUpgrade.slideSubmitLoading = false;
						}
					}
				}).catch(function(error){})
			},
			productFmt(row,column,cellValue,index){
				var code = {
		    			'CPE_VERSION' : 'ODU',
		    			'ODU' : 'ODU',
		    			'CPE_IDU_VERSION' : 'IDU',
		    			'IDU' : 'IDU'
		    	}
		    	return code[cellValue]
			},
		},
		watch:{
			"importForm.moduleName":function(){
				this.$refs.cpeFileImportForm.validateField("version");
			}
		},
		mounted(){
			eventBus.$off('cpe-upgrade-init').$on('cpe-upgrade-init',this.init);
			eventBus.$off('cpe-upgrade-importSubmit').$on('cpe-upgrade-importSubmit',this.submit);
			eventBus.$off('cpe-upgrade-cancelImport').$on('cpe-upgrade-cancelImport',this.cancel);
		}
	}) 
</script>