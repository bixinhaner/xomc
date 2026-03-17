<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#egwImportfile .el-form-item__label{
	line-height:26px;
	width:140px;
	text-align:left;
}
#egwImportfile .el-form-item__error{
	white-space: nowrap;
}
#egwImportfile .el-form-item{
	margin-bottom:30px;
}
#egwImportfile .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#egwImportfile .uploadInput .el-input__suffix{
	top:5px !important;
}

.item-flex .el-form-item{
	display: inline-block;
	width: 400px;
	position: relative;
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

<div id='egwImportfile'>
	<el-form ref="egwFileImportForm" :model="importForm" label-width="120px" label-position="left" :rules="importRules" style="margin-left:30px;margin-top:30px;" :hide-required-asterisk=true>
		<div class="item-flex">
			<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
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
					<el-input style="width:300px;" class="uploadInput" :readonly="true" :value='importForm.fileName' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>								
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
				<el-input v-show="operationType != 'add'" v-model='importForm.fileName' :disabled="viewFlag || editFlag" style="width:300px;"></el-input>
				<div class="filePrompt"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><%=rb.getString("RPMWenJianTiShi")%></div>
			</el-form-item>
			<el-form-item label="<%=rb.getString("BanBen")%>" prop="version" style="margin-left:300px;">
				<el-input v-model='importForm.version' :disabled="viewFlag" style="width:200px;"></el-input>
			</el-form-item>
		</div>
		<div class="item-flex">
			<!--<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
				<el-select v-model='importForm.recommend' :disabled="viewFlag">
					<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
					<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
				</el-select>
			</el-form-item>-->
			<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" >
				<el-input v-model='importForm.productType' :disabled="true" style="width:200px;"></el-input>
			</el-form-item>
		</div>
	
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='description'>
			<el-input :disabled="viewFlag" v-model='importForm.description' type='textarea' :rows='3' style='width:600px;'></el-input>
		</el-form-item>
	</el-form>
</div>
<script>
	new Vue({
		el:'#egwImportfile',
		data(){
			var vm = this;
			var versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingShuRuWenJianBanBen")%>');
				}
			};
			var fileNameValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
					}else if(!fileFormatMatch(value,"rpm")){
						callback('<%=rb.getString("ZhIZhiChiRPMWenJian")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			};
		
			return{
				importForm:{
					productType:'eGW',
					fileName:'',
					version:'',
					// recommend:'1',
					description:'',
				},
				importRules:{
					version:[
						{validator: versionValidate}
					],
					fileName:[
						{validator: fileNameValidate}
					],
					productType:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					],
					// recommend:[
					// 	{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					// ],
					
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
				
				fileErrorData:'',
				defaultVersion:''
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
						fileId:vm.fileId
					};
				axios.post('${ctx}/egw/softwareFile/getFileInfo.action',stringify(params)).then(function(response){
						var data = response.data;
						// ['productType','fileName','version','recommend','description',]
						['productType','fileName','version','description',].map((item)=>{
							vm.importForm[item] = data[item];
							vm.defaultVersion = data.version;
						})
						initForm(vm.$refs.egwFileImportForm);
					}).catch(function(error){})
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
				var vm = this;
				vm.importForm.fileName = file.name;
				vm.importForm.version = file.name.substring(0,file.name.lastIndexOf("."));
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
				fd.append('fileSize',fileSize);//文件大小
				fd.append('description',vm.importForm.description);//描述
				fd.append('productType',vm.importForm.productType);
				fd.append('version',vm.importForm.version);
				// fd.append('recommend',vm.importForm.recommend);
				vm.fileErrorData = vm.$refs.upload.uploadFiles[0];
				$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
				$("#winUploadPro").window("open");// 打开进度条窗口
                egwUpgrade.slideSubmitLoading = true;
				axios.post("${ctx}/egw/softwareFile/uploadSoftwareFile.action",fd,config).then(function(response){
					var data = response.data
					$("#winUploadPro").window("close");// 关闭进度条窗口
					if(data["success"]){
						vm.fileErrorData = '';
						$.messager.alert('<%=rb.getString("TiShi")%>','<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["message"]);
						eventBus.$emit('hide-egwUpgrade-slide');
					}else{
						vm.$message.error(data["message"]);
                        egwUpgrade.slideSubmitLoading = false;
					}
				})
				
				return false;
			},
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.egwFileImportForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-egwUpgrade-slide');
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-egwUpgrade-slide');
				}
			},
			submit(){
				var vm = this;
				// 防止多次提交
                if(egwUpgrade.slideSubmitLoading)return

				vm.$refs.egwFileImportForm.validate((valid) => {
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
						id:vm.fileId,
						fileName:vm.importForm.fileName,
						productType:vm.importForm.productType,
						version:vm.importForm.version,
						// recommend:vm.importForm.recommend,
						description:vm.importForm.description
					};
                egwUpgrade.slideSubmitLoading = true;
				axios.post('${ctx}/egw/softwareFile/editSoftwareFileInfos.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							eventBus.$emit('hide-egwUpgrade-slide');
						}else{
							vm.$message.error(data["message"]);
                            egwUpgrade.slideSubmitLoading = false;
						}
					}
				}).catch(function(error){})
			},
		},
		mounted(){
			eventBus.$off('egw-upgrade-init').$on('egw-upgrade-init',this.init);
			eventBus.$off('egw-upgrade-importSubmit').$on('egw-upgrade-importSubmit',this.submit);
			eventBus.$off('egw-upgrade-cancelImport').$on('egw-upgrade-cancelImport',this.cancel);
		}
	}) 
</script>