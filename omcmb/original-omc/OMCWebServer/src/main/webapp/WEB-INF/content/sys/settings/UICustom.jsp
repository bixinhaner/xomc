<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
#customerUI .labelSty{
	display:inline-block;
	width:150px;
	vertical-align:top;
}
#customerUI .splitGroup_item {
	margin-bottom: 0px;
}
#customerUI .splitGroup_item .el-icon-plus {
	margin-top:60px;
	font-size:24px;
	color:#333;
}
#customerUI .disable .el-upload.el-upload--picture-card ,
#customerUI .disable .el-button--success.is-plain {
	display:none !important;
}
.file-tips {
	color: #7a7992;
}
</style>
	
<div class="panelDefault" id='customerUI' style="border:none; ">
	<div class="splitGroup" style='height:calc(100% - 80px); overflow: auto; padding: 30px 0 0 ;'>
		<div class="splitGroup_title">
			<span class=""></span>
			<span><%=rb.getString("UIDingZhiHua")%></span>
		</div>
		<div class="splitGroup_body">
			<el-form ref="form" :model="eqForm" :rules="rules" label-position="left" label-width="170">
				<el-form-item label='<%=rb.getString("OMCMingCheng")%>'>
					<el-input v-model="systemName"></el-input>
				</el-form-item>
				<el-form-item label='<%=rb.getString("ZhuTiSe")%>'>
					<el-color-picker v-model="colorUI"></el-color-picker>
					<el-button @click="preView">Preview</el-button>
				</el-form-item>
				<el-form-item label='<%=rb.getString("DengLuBeiJing")%>' prop="image1Size" style="margin-bottom: 20px;">
					<el-upload ref="upload1" action="#" list-type="picture-card" accept="image/jpeg,image/jpg,image/png" :limit="1" class="splitGroup_item"
						style="display:inline-block;" :class="{disable:eqObj.uploadDisabled1}" 
						:on-change="handleChange1" :on-remove="handleRemove1" :before-upload="beforeImageUpload1">
						<i slot="default" class="el-icon el-icon-plus" ></i>
						<div slot="file" slot-scope="{file}">
							<img class="el-upload-list__item-thumbnail" :src="file.url" >
						</div>
					
					</el-upload>
					<div class="file-tips">
						<i class="el-icon el-icon-message"></i> <%=rb.getString("TiShi1M")%>
					</div>
					<el-input v-model="eqForm.image1Size" type="hidden"></el-input>
				</el-form-item>
				
				<el-form-item label='<%=rb.getString("LogoSmall")%>' prop="image2Size" style="margin-bottom: 20px;">
					<el-upload ref="upload2" action="#" list-type="picture-card" accept="image/jpeg,image/jpg,image/png" :limit="1" class="splitGroup_item"
						style="display:inline-block;" :class="{disable:eqObj.uploadDisabled2}"
						:on-change="handleChange2"  :on-remove="handleRemove2" :before-upload="beforeImageUpload2">
						<i slot="default" class="el-icon el-icon-plus" ></i>
						<div slot="file" slot-scope="{file}">
							<img class="el-upload-list__item-thumbnail" :src="file.url" >
						</div>
					
					</el-upload>
					<div class="file-tips">
						<i class="el-icon el-icon-message"></i> <%=rb.getString("TiShi4KB")%>
					</div>
					<el-input v-model="eqForm.image2Size" type="hidden"></el-input>
				</el-form-item>
				
				<el-form-item label='<%=rb.getString("LogoBig")%>' prop="image3Size">
					<el-upload ref="upload3" action="#" list-type="picture-card" accept="image/jpeg,image/jpg,image/png" :limit="1" class="splitGroup_item"
						style="display:inline-block;" :class="{disable:eqObj.uploadDisabled3}"
						:on-change="handleChange3"  :on-remove="handleRemove3" :before-upload="beforeImageUpload3">
						<i slot="default" class="el-icon el-icon-plus" ></i>
						<div slot="file" slot-scope="{file}">
							<img class="el-upload-list__item-thumbnail" :src="file.url" >
						</div>
					
					</el-upload>
					<div class="file-tips">
						<i class="el-icon el-icon-message"></i> <%=rb.getString("TiShi4KB")%>
					</div>
					<el-input v-model="eqForm.image3Size" type="hidden"></el-input>
				</el-form-item>
			</el-form>
		</div>
	
	</div>
	
	<div class="" style='padding: 13px 30px;'>
		<el-button type="primary" @click='submit'><%=rb.getString("QueDing")%></el-button>
		<el-button @click="restoreUI"><%=rb.getString("HuiFu")%></el-button>
	</div>

</div>


<script type="text/javascript">
	//创建vue实例
	var customerUIVue = new Vue({
		el: '#customerUI',
		data() {
			var vm = this,
				validBgSize = function(rule, value, cb) {
					// 1M == 1048576Bit
					if(value - 1048576 > 0) {
						cb('<%=rb.getString("WenJianDaXiaoZuiDa1M")%>')
					}else {
						cb()
					}
				},
				validSmallSize = function(rule, value, cb) {
					// 1kb == 1024Bit
					if(value - 409600 > 0) {
						cb('<%=rb.getString("WenJianDaXiaoZuiDa4KB")%>')
					}else {
						cb()
					}
				},
				validBigSize = function(rule, value, cb) {
					// 1kb == 1024Bit
					if(value - 409600 > 0) {
						cb('<%=rb.getString("WenJianDaXiaoZuiDa4KB")%>')
					}else {
						cb()
					}
				};

			return {
			    systemName:'',
				colorUI:"",
				colorUIRgb:"(25,19,187)",
				eqObj:{
					uploadDisabled1:false,
					uploadDisabled2:false,
					uploadDisabled3:false
				},
				eqForm:{
					image1:'',
					image2:'',
					image3:'',
					image1Size:'',
					image2Size:'',
					image3Size:''
				},
				rules: {
					image1Size: [{validator: validBgSize}],
					image2Size: [{validator: validSmallSize}],
					image3Size: [{validator: validBigSize}]
				}
			}
		},
		computed: {
		},
		mounted() {
			var vm = this;
			this.init();
				
		},
		methods: {
			// 初始化函数
			init() {
				var vm = this;
				
				vm.colorUI = uiCustom.ui_color;
				vm.systemName = document.title;
					
			},
			handleChange1(file,fileList){
				if(fileList.length ==1){
					this.eqObj.uploadDisabled1 =  true;
					this.$set(this.eqObj,'uploadDisabled1',true);
				
				}else {
					this.eqObj.uploadDisabled1 =  false;
					this.$set(this.eqObj,'uploadDisabled1',false);
				}
				this.$forceUpdate();
				
			},
			handleRemove1(file,fileList){
				this.eqObj.uploadDisabled1 =  false;
				this.eqForm.image1 = "";
				this.eqForm.image1Size = "";
				this.$forceUpdate();
			},
			beforeImageUpload1(file){
				var vm = this;
				vm.eqForm.image1Size = file.size;
				return new Promise(function(resolve,reject){
					var reader = new FileReader();
					reader.readAsDataURL(file)
					reader.onload = function(event){
						vm.eqForm.image1 = event.target.result
					}
				})
			},
			handleChange2(file,fileList){
				if(fileList.length ==1){
					this.eqObj.uploadDisabled2 =  true;
					this.$set(this.eqObj,'uploadDisabled2',true);
				
				}else {
					this.eqObj.uploadDisabled2 =  false;
					this.$set(this.eqObj,'uploadDisabled2',false);
				}
				this.$forceUpdate();
				
			},
			handleRemove2(file,fileList){
				this.eqObj.uploadDisabled2 =  false;
				this.$forceUpdate();
				this.eqForm.image2 = "";
				this.eqForm.image2Size = "";
			},
			beforeImageUpload2(file){
				var vm = this;
				vm.eqForm.image2Size = file.size;
				return new Promise(function(resolve,reject){
					var reader = new FileReader();
					reader.readAsDataURL(file)
					reader.onload = function(event){
						vm.eqForm.image2 = event.target.result
					}
				})
			},
			handleChange3(file,fileList){
				if(fileList.length ==1){
					this.eqObj.uploadDisabled3 =  true;
					this.$set(this.eqObj,'uploadDisabled3',true);
				
				}else {
					this.eqObj.uploadDisabled3 =  false;
					this.$set(this.eqObj,'uploadDisabled3',false);
				}
				this.$forceUpdate();
				
			},
			handleRemove3(file,fileList){
				this.eqObj.uploadDisabled3 =  false;
				this.$forceUpdate();
				this.eqForm.image3 = "";
				this.eqForm.image3Size = "";
			},
			beforeImageUpload3(file){
				var vm = this;
				vm.eqForm.image3Size = file.size;
				return new Promise(function(resolve,reject){
					var reader = new FileReader();
					reader.readAsDataURL(file)
					reader.onload = function(event){
						vm.eqForm.image3 = event.target.result
					}
				})
			},
			// 颜色值设置预览 
			preView(){
				var vm = this;
				//var root = document.querySelector(":root");
				
				document.documentElement.style.setProperty("--main-color" , vm.colorUI);
				
				vm.colorUIRgb = changeColor(vm.colorUI);

				document.documentElement.style.setProperty("--main-color-rgba1" , vm.colorUIRgb);
				
			},
			//保存当前设置颜色和图片 
			submit(){
				var vm = this;
				
				 var params = {
					"ui_color":vm.colorUI,
					"ui_login_background":vm.eqForm.image1 ? vm.eqForm.image1 :uiCustom.ui_login_background ,
					"ui_menu_logo_up": vm.eqForm.image2 ? vm.eqForm.image2 :uiCustom.ui_menu_logo_up,
					"ui_menu_logo_down": vm.eqForm.image3 ? vm.eqForm.image3 :uiCustom.ui_menu_logo_down,
					"ui_restore":"false",
					"ui_omc_name":vm.systemName
				}
				
				vm.$refs.form.validate(function(valid){
					if(valid) {
						axios.post('${ctx}/ui/customization/updateCustomizationInfo.action', stringify(params)).then(function (response) {
							let data = response.data;
							if (data.success) {
								vm.$message({
									message: data.message || '<%=rb.getString("ChengGong")%>',
									type: 'success',
								});
								document.documentElement.style.setProperty("--main-bg" ,'url('+ params.ui_login_background +')');
								document.documentElement.style.setProperty("--main-color" ,params.ui_color);
								document.documentElement.style.setProperty("--logo-small" , 'url('+ params.ui_menu_logo_up +')');
								document.documentElement.style.setProperty("--logo-big" ,'url('+ params.ui_menu_logo_down +')');
								document.documentElement.style.setProperty("--logo-big-white" ,'url('+ params.ui_menu_logo_down +')');
								
								vm.colorUIRgb = changeColor(vm.colorUI);

								document.documentElement.style.setProperty("--main-color-rgba1" , vm.colorUIRgb);
								document.title = params.ui_omc_name;
								updateFavicon(vm.colorUI);
							} else {
								vm.$message.error(data.message)
							}

						}).catch(function (error) { }) 
					}
				})
			},
			//恢复默认颜色值和背景图等 
			restoreUI(){
				var vm = this;
				
				var confirmStr = '<%=rb.getString("QueDingHuiFuMoRenUI")%> ';
				vm.$confirm(confirmStr, '<%=rb.getString("QueRen")%>', {
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
					type: 'warning',
					closeOnClickModal: false
				}).then(() => {
					var url = "${ctx}/ui/customization/updateCustomizationInfo.action";
					
					axios.post(url, stringify(uiCustomOld)).then(function (res) {
						var data = res.data;
						if(data.success){
							document.documentElement.style.setProperty("--main-bg" ,'url('+ uiCustomOld.ui_login_background +')');
							document.documentElement.style.setProperty("--main-color" ,uiCustomOld.ui_color);
							document.documentElement.style.setProperty("--logo-small" , 'url('+ uiCustomOld.ui_menu_logo_up +')');
							document.documentElement.style.setProperty("--logo-big" ,'url('+ uiCustomOld.ui_menu_logo_down +')');
							document.documentElement.style.setProperty("--logo-big-white" ,'url('+ uiCustomOld.ui_menu_logo_down +')');
							vm.colorUI = uiCustomOld.ui_color;
							document.title = uiCustomOld.ui_omc_name;
							vm.systemName = uiCustomOld.ui_omc_name;

							vm.$refs.upload1.clearFiles();
							vm.$refs.upload2.clearFiles();
							vm.$refs.upload3.clearFiles();
							vm.eqObj.uploadDisabled1 =  false;
							vm.eqObj.uploadDisabled2 =  false;
							vm.eqObj.uploadDisabled3 =  false;
							vm.eqForm.image1 = "";
							vm.eqForm.image1Size = "";
							vm.eqForm.image2 = "";
							vm.eqForm.image2Size = "";
							vm.eqForm.image3 = "";
							vm.eqForm.image3Size = "";

							Object.assign(uiCustom, {
								"ui_color": "#FF4614",
								"ui_login_background": "./images/login/login_bg.png",
								"ui_menu_logo_up": "./images/login/nav_logo_collapse.png",
								"ui_menu_logo_down": "./images/login/logo_big.png",
								"ui_restore": "true",
								"ui_omc_name": "BaiOMC"
							});
							vm.colorUIRgb = changeColor(vm.colorUI);
							document.documentElement.style.setProperty("--main-color-rgba1" , vm.colorUIRgb);
							updateFavicon(vm.colorUI);
						}else {
							vm.$message.error(data.message)
						}
						
					})
					
				}).catch(() => {
				})
				

			}
		}
	})
</script>