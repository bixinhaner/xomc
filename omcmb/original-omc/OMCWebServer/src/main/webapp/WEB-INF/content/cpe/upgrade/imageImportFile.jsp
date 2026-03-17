<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#imageImportfile .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#imageImportfile .el-form-item__error{
	white-space: nowrap;
}
#imageImportfile .el-form-item{
	margin-bottom:30px;
}
#imageImportfile .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#imageImportfile .uploadInput .el-input__suffix{
	top:5px !important;
}

.item-flex .el-form-item{
	display: inline-block;
	width: 400px;
	position: relative;
}
.productModelInputCls{
	position: absolute;
	left: 1px;
	top: 2px;
	width: 170px;
}
.productModelInputCls .el-input__inner{
	border:none;
	height: 24px;
}
.freq-item{
	padding-top: 10px;
}
.freq-item-suffix {
	display: inline-block;
	margin: 2px;
	padding: 0px 5px;
	border: 1px solid #A8C9FA;
	background: #e6f5fa;
}
.freq-item-suffix .num-item {
	display: inline-block;
	height: 20px;
	min-width: 65px;
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

<div id='imageImportfile'>
	<el-form ref="cpeFileImportForm" :model="importForm" label-width="120px" label-position="left" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<div class="item-flex">
			<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="file_name">
				<el-upload 
					ref="upload"
					v-show="operationType == 'add'" 
					:before-upload='beforeUpload' 
					:on-success='checkFile' 
					:on-change="fileChange"  
					:show-file-list=false 
					:action="uploadFileUrl" 
					:data="fileParams" 
					name="uploadFile" 
					:auto-upload="false">
					<el-input style="width:300px;" class="uploadInput" :readonly="true" :value='importForm.file_name' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>								
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
				<el-input v-show="operationType != 'add'" v-model='importForm.file_name' :disabled="viewFlag || editFlag" style="width:300px;"></el-input>
				<div class="filePrompt"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><%=rb.getString("TGZBINWenJianTiShi")%></div>
			</el-form-item>
			<el-form-item label="<%=rb.getString("BanBen")%>" prop="version" style="margin-left:300px;">
				<el-input v-model='importForm.version' :disabled="viewFlag" style="width:200px;"></el-input>
			</el-form-item>
		</div>
		<div class="item-flex">
			<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
				<el-select v-model='importForm.recommend' :disabled="viewFlag">
					<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
					<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
				</el-select>
			</el-form-item>
			<!--<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style="margin-left:300px;">
				<el-select v-model='importForm.product' :disabled="viewFlag">
					<el-option label='ODU' value='CPE_VERSION'></el-option>
					<el-option label='IDU' value='CPE_IDU_VERSION'></el-option>
				</el-select>
			</el-form-item>-->
		</div>
		<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='toWho' v-if="showToWho">
			<el-select v-model='importForm.toWho'>
				<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
				<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
				<el-option label='<%=rb.getString("BetaBanBen")%>' value="beta"></el-option>
			</el-select>
		</el-form-item>
		<div class="tableBox">
			<div class="tableTitleCls">
				<span><%=rb.getString("ChanPinXingHao")%></span>
			</div>
			<div class="addModuleNameBox" v-if="showPairgrid">
				<span class="el-icon el-icon-circle-add" @click="addModuleNameClick"></span>
			</div>
			<el-form-item label='' label-width="0px" prop='model_name' style="position: relative;">
				<el-pairgrid v-if="showPairgrid" @checked-change='moduleNameSelectChange'   ref="moduleNamePairgrid" :rownumber="true" :right-url="moduleNameRightUrl" :left-url="moduleNameLeftUrl" :height="height" :readonly='viewFlag' row-key="model_name" :query-params="queryModuleNameForm" :title="moduleNameTitle" :messages="{placeholder:'<%=rb.getString("ChanPinXingHao")%>'}">
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column prop='model_name' sortable label='<%=rb.getString("ChanPinXingHao")%>'></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryModuleNameList" placeholder="<%=rb.getString("ChanPinXingHao")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='model_name' label='<%=rb.getString("ChanPinXingHao")%>'></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable v-else id="'selected_modelName_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="moduleNameRightUrl" :height="height" front-pagination="true" pagination="true">
					<el-table-column prop='model_name' label='<%=rb.getString("ChanPinXingHao")%>'></el-table-column>
				</el-ctable>
			</el-form-item>
		</div>
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
			<el-input :disabled="viewFlag" v-model='importForm.desc' type='textarea' :rows='3' style='width:600px;'></el-input>
		</el-form-item>
	</el-form>
	<el-dialog title='<%=rb.getString("TianJia")%>' id="addModuleNameDialog" :visible.sync="showAddModuleNameDialog" top="30vh" ref="addModuleNameDialog" :width="addModuleNameDialogWidth" 
			:close-on-click-modal="false"  @close='closeAddModuleName' append-to-body >
			<el-form  
				:model="addModuleNameForm"
				ref="addModuleNameFormBox" 
				:rules="addModuleNameRules" 
				label-position="left" 
				id="addModuleNameForm"
				@submit.native.prevent
			>
				<el-form-item  style="margin-left:40px;margin-top:10px;" label="<%=rb.getString("ChanPinXingHao")%>" prop='addModuleName'  label-width="120px">
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
		el:'#imageImportfile',
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
							axios.get("${ctx}/cell/version/verifyUVExist.action",{
								params:{
									fileType : 'upgradecpe',
									version : value,
									//productType:vm.importForm.product == 'CPE_VERSION' ? 'ODU' : 'IDU',
									productModel:vm.importForm.model_name
								}
							}).then(function(response){
								var data = response.data;
								if(data){
									callback("<%=rb.getString("Msg_ShengJiWenJianBanBenYiJingCunZai")%>");
								}else{
									callback();
								}
							})
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
					}else if(!fileFormatMatch(value,"bin,tgz")){
						callback('<%=rb.getString("ZhIZhiChiTGZBINWenJian")%>');
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
			var validatorModuleName = function(rule,value,callback) {
				if(this.operationType == 'view'){
					callback();
				}else{
					if(value) {
						callback();
					}else {
						callback('<%=rb.getString("QingXuanZeChanPinXingHao")%>');
					}
				}
				
			};
			return{
				importForm:{
					//product:'CPE_VERSION',
					file_name:'',
					version:'',
					recommend:'1',
					model_name:'',
					toWho:'all',
					desc:'',
				},
				importRules:{
					version:[
						{validator: versionValidate}
					],
					file_name:[
						{validator: fileNameValidate}
					],
					// product:[
					// 	{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					// ],
					recommend:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					],
					model_name:[
						{validator:validatorModuleName,trigger:'change'}
					],
				},
				showPairgrid:true,
				viewFlag : false,
				editFlag:false,
				fileParams:{},            //上传文件时自定义的参数  
				fileList:[],
				uploadFileUrl:'',
				operationType:'add',
				height:'370px',
				fileId:'',
				moduleNameLeftUrl:'${ctx}/cell/CPE/queryModelNames4ImageFileImport.action',
				moduleNameRightUrl:'',
				moduleNameTitle:['','<%=rb.getString("YiXuanLieBiao")%>'],
				queryModuleNameForm:{
					searchText:'',
				},
				addModuleNameForm:{
					addModuleName:''
				},
				addModuleNameRules:{
					addModuleName:[
						{validator: addModuleNameValidate}
					],
				},
				showAddModuleNameDialog:false,
				addModuleNameDialogWidth:'460px',
				fileErrorData:'',
				defaultVersion:'',
                multipartMaxFileSize: multipartMaxFileSize
			}
		},
		computed:{
			showToWho(){
            	return isCloudCore == 'true'? true : false;
        	},
    	},
		methods:{
			// 初始化
			init(type,id){
				var vm = this;
				vm.operationType = type;
				vm.fileId = id;
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
						versionId:vm.fileId
					};
				axios.post('${ctx}/cell/version/getDeviceVersionFileInfo.action',stringify(params)).then(function(response){
						var data = response.data;
						['file_name','version','recommend','toWho','desc','model_name'].map((item)=>{
							vm.importForm[item] = data[item];
							vm.defaultVersion = data.version;
						})
						vm.moduleNameRightUrl = '${ctx}/cell/version/getSelectedModelName4CpeUpgradeFileInfo.action?versionId='+vm.fileId;
						initForm(vm.$refs.cpeFileImportForm);
					}).catch(function(error){})
			},
			// 模块名称列表选择事件
			moduleNameSelectChange(){
				var vm =this,
					data = vm.$refs.moduleNamePairgrid.getData();
					selectedModuleName = '',
					selectedModuleNameList=[];
				if(data.length != 0){
					data.map(function(item){
						selectedModuleNameList.push(item.model_name)
					})
				}
				selectedModuleName = selectedModuleNameList.join(',');
				vm.importForm.model_name = selectedModuleName;
			},
			// 模块名称列表 模糊搜索
			queryModuleNameList(val){
				var vm = this;
				vm.queryModuleNameForm.searchText = val;
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
						vm.$refs.moduleNamePairgrid.appendCheckedRow({model_name:vm.addModuleNameForm.addModuleName});
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
			/**
			* 文件上传成功函数 
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			*/
			checkFile(res,file){    //发送请求，校验device文件内容 
				
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
                    vm.importForm.file_name = file.name;
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
				vm.importForm.file_name = '';
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
				// if(vm.importForm.product == 'CPE_VERSION'){
				// 	fileType = 'upgradecpe';
				// }else{
				// 	fileType = 'upgradeiducpe';
				// }
				fd.append('uploadFile',file); //文件流
				fd.append('newFileName',fileName);//文件名
				fd.append('fileSize',fileSize);//文件大小
				//fd.append('fileType',fileType); // 文件类型
				fd.append('desc',vm.importForm.desc);//描述
				//fd.append('product',vm.importForm.product);
				fd.append('model_name',vm.importForm.model_name);
				fd.append('version',vm.importForm.version);
				fd.append('recommend',vm.importForm.recommend);
				fd.append('to_who',vm.importForm.toWho);
				fd.append('fileType','upgradecpe');
				vm.fileErrorData = vm.$refs.upload.uploadFiles[0];
				$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
				$("#winUploadPro").window("open");// 打开进度条窗口
                cpeUpgrade.slideSubmitLoading = true;
				axios.post("${ctx}/cell/version/uploadVersionFile.action",fd,config).then(function(response){
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
						versionId:vm.fileId,
						fileType:'',
						fileName:vm.importForm.file_name,
						//product:vm.importForm.product,
						version:vm.importForm.version,
						model_name:vm.importForm.model_name,
						toWho:vm.importForm.toWho,
						recommend:vm.importForm.recommend,
						desc:vm.importForm.desc
					};
				// if(vm.importForm.product == 'CPE_VERSION'){
				// 	params.fileType = '3';
				// }else{
				// 	params.fileType = '4';
				// }
                cpeUpgrade.slideSubmitLoading = true;
				axios.post('${ctx}/cell/version/goModifyDeviceVersionFileInfo.action',stringify(params)).then(function(response){
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
		},
		mounted(){
			eventBus.$off('cpe-upgrade-init').$on('cpe-upgrade-init',this.init);
			eventBus.$off('cpe-upgrade-importSubmit').$on('cpe-upgrade-importSubmit',this.submit);
			eventBus.$off('cpe-upgrade-cancelImport').$on('cpe-upgrade-cancelImport',this.cancel);
		}
	}) 
</script>