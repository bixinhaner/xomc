<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#middleImportfile .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#middleImportfile .el-form-item__error{
	white-space: nowrap;
}
#middleImportfile .el-form-item{
	margin-bottom:30px;
}
#middleImportfile .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#middleImportfile .uploadInput .el-input__suffix{
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
.modelNameInputCls{
	position:absolute;
	left:1px;
	top: 1px;
	z-index:999;
}
.modelNameInputCls .el-input__inner{
	border: none;
	height: 24px;
}
</style>

<div id='middleImportfile'>
	<el-form ref="cpeFileImportForm" :model="importForm" label-width="120px" label-position="left" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<div class="item-flex">
			<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
				<el-upload v-show="operationType == 'add'" :before-upload='beforeUpload' :on-success='checkFile' :on-change="fileChange"  :show-file-list=false ref="upload"
						:action="uploadFileUrl" :data="fileParams" name="uploadFile" :auto-upload="false">
					<el-input style="width:300px;" class="uploadInput" :readonly="true" :value='importForm.fileName' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>								
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
				<el-input v-show="operationType != 'add'" v-model='importForm.fileName' :disabled="viewFlag || editFlag" style="width:300px;"></el-input>
				<div class="filePrompt"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><%=rb.getString("TGZBINWenJianTiShi")%></div>
			</el-form-item>
			<el-form-item label="<%=rb.getString("BanBen")%>" prop="version" style="margin-left:200px;">
				<el-input v-model='importForm.version' :disabled="viewFlag || editFlag " style="width:200px;"></el-input>
			</el-form-item>
		</div>
		<el-form-item label='<%=rb.getString("ChanPinXingHao")%>' prop='modelName' style="position: relative;">
			<div class="modelNameInputCls">
				<el-input v-model='importForm.modelName' :disabled="viewFlag" style="width:170px;"></el-input>
			</div>
			<el-select v-model='importForm.modelName' :disabled="viewFlag" filterable>
				<el-option v-for="item in productModelData" :label="item.model_name" :value="item.model_name"></el-option>
			</el-select>
		</el-form-item>
		<div class="tableBox">
			<div class="tableTitleCls">
				<span><%=rb.getString("ChuShiBanBen")%></span>
				<span></span>
			</div>
			<el-form-item label='' label-width="0px" prop='oriVersion' style="position: relative;">
				<el-pairgrid v-if="showPairgrid" @checked-change='oriVersionSelectChange' :limit="9"   ref="oriVersionPairgrid" :rownumber="true" :right-url="oriVersionRightUrl" :left-url="oriVersionLeftUrl" :height="height" :readonly='viewFlag' row-key="software_version" :query-params="queryOriVersionForm" :title="oriVersionTitle" :messages="{placeholder:'<%=rb.getString("ChuShiBanBen")%>'}">
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column prop='software_version' label='<%=rb.getString("ChuShiBanBen")%>'></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryOriVersionList" placeholder="<%=rb.getString("ChuShiBanBen")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='software_version' label='<%=rb.getString("ChuShiBanBen")%>'></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable v-else id="selected_oriVersion_list" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="oriVersionRightUrl" :height="height" front-pagination="true" pagination="true">
					<el-table-column prop='software_version' label='<%=rb.getString("ChuShiBanBen")%>'></el-table-column>
				</el-ctable>
			</el-form-item>
		</div>
		
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='description'>
			<el-input :disabled="viewFlag" v-model='importForm.description' type='textarea' :rows='3' style='width:600px;'></el-input>
		</el-form-item>
	</el-form>
	
</div>
<script>
	new Vue({
		el:'#middleImportfile',
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
							axios.get("${ctx}/cell/version/verifyMidUVExist.action",{
								params:{
									modelName : vm.importForm.modelName,
									version : value
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
					}else{
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
			var validatorOriVersion = function(rule,value,callback) {
				if(this.operationType == 'view'){
					callback();
				}else{
					if(value) {
						callback();
					}else {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}
				}
				
			};
			var validatorModelName = function(rule,value,callback) {
				if(value) {
					callback();
				}else {
					callback('<%=rb.getString("QingXuanZeChanPinXingHao")%>');
				}
			};
			return{
				importForm:{
					product:'CPE_VERSION',
					fileName:'',
					version:'',
					modelName:'',
					oriVersion:'',
					description:'',
				},
				
				productModelData:[],
				
				queryOriVersionForm:{
					version:'',
				},
				height:'370px',
				
				oriVersionLeftUrl:'${ctx}/cell/CPE/queryCpeCurrVersions.action',
				oriVersionRightUrl:'',
				oriVersionTitle:['','<%=rb.getString("YiXuanLieBiao")%>'],
				importRules:{
					version:[
						{validator: versionValidate}
					],
					fileName:[
						{validator: fileNameValidate}
					],
					modelName:[
						{validator:validatorModelName,trigger:'change'}
					],
					oriVersion:[
						{validator:validatorOriVersion,trigger:'change'}
					]
				},
				showPairgrid:true,
				viewFlag : false,
				editFlag:false,
				fileParams:{},            //上传文件时自定义的参数  
				fileList:[],
				uploadFileUrl:'',
				operationType:'add',
				fileVersion:'',
				fileModelName:'',
				fileErrorData:'',
				defaultVersion:'',
                multipartMaxFileSize: multipartMaxFileSize
			}
		},
		computed:{},
		methods:{
			// 初始化
			init(type,version,modelName){
				var vm = this;
				vm.operationType = type;
				vm.fileVersion = version;
				vm.fileModelName = modelName;
				vm.getProductModelData();
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
						midVersion:vm.fileVersion,
						modelName:vm.fileModelName
					};
				axios.post('${ctx}/cell/version/queryMidFileInfo.action',stringify(params)).then(function(response){
						var data = response.data;
						['fileName','version','description','modelName','oriVersion'].map((item)=>{
							vm.importForm[item] = data[item];
							vm.defaultVersion = data.version;
						})
						vm.oriVersionRightUrl = '${ctx}/cell/version/queryMidFileSelectedOriVersionInfo.action?midVersion='+vm.fileVersion +'&modelName='+ vm.fileModelName;
						initForm(vm.$refs.cpeFileImportForm);
					}).catch(function(error){})
			},
			// 目标版本列表选择事件
			oriVersionSelectChange(){
				var vm =this,
					data = vm.$refs.oriVersionPairgrid.getData();
					selectedOriVersion = '',
					selectedOriVersionList=[];
				if(data.length != 0){
					data.map(function(item){
						selectedOriVersionList.push(item.software_version)
					})
				}
				selectedOriVersion = selectedOriVersionList.join(',');
				vm.importForm.oriVersion = selectedOriVersion;
			},
			//目标版本列表 模糊搜索
			queryOriVersionList(val){
				var vm = this;
				vm.queryOriVersionForm.version = val;
			},
			// 获取产品型号下拉数据
			getProductModelData(){
				var  vm = this;
				axios.post('${ctx}/cell/CPE/queryModelNames.action').then(function(response){
					var data = response.data;
					vm.productModelData = data;
				}) 
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
				fd.append('modelName',vm.importForm.modelName);
				fd.append('version',vm.importForm.version);
				fd.append('oriVersion',vm.importForm.oriVersion);
				vm.fileErrorData = vm.$refs.upload.uploadFiles[0];
				$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
				$("#winUploadPro").window("open");// 打开进度条窗口
                cpeUpgrade.slideSubmitLoading = true;
				axios.post("${ctx}/cell/version/uploadMidVersionFile.action",fd,config).then(function(response){
					var data = response.data
					$("#winUploadPro").window("close");// 关闭进度条窗口
					if(data["MD5"]){
						$.messager.alert(TiShi,'<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["MD5"]);
						eventBus.$emit('hide-cpeUpgrade-slide');
					}else{
						vm.fileErrorData = '';
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
						midVersion:vm.importForm.version,
						modelName:vm.importForm.modelName,
						oriVersion:vm.importForm.oriVersion,
						desc:vm.importForm.description
					};
                cpeUpgrade.slideSubmitLoading = true;
				axios.post('${ctx}/cell/version/updateMidVersionInfo.action',stringify(params)).then(function(response){
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
		watch:{
			"importForm.modelName":function(){
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