<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#upgradeOperDiv .el-input{
		width:300px;
	}
	#upgradeOperDiv .queryGroup{
		height:26px;
		margin-left:0px;
		border-radius:0px;
		padding:0px 15px 0px 0px;
	}
	#upgradeOperDiv .queryGroup .el-input{
		width:265px;
	}
	#upgradeOperDiv .queryGroup input{
		width:250px;
		height:24px;
	}
	.oriVersionClass{
		height:0px;
		overflow:hidden;
		position:absolute;
		left:0px;
		top:45px;
	}
	.oriVersionClass .el-select__tags{
		max-height:0px;
		overflow:hidden;
	}
	.oriVersionClass .el-input {
		max-height: 0px;
	}
	#upgradeOperDiv .el-icon-down:before{
		color:#c0c4cc;
	}
	.suffixItem{
		display:inline-block;
		margin-bottom:0px;
		margin-right:5px;
	}
	.suffixItem .el-form-item__content{
		line-height:16px;
	}
	.form-suffix{
		margin-top:5px;
	}
	.form-suffix .text{
		overflow:hidden;
		white-space:nowrap;
		text-overflow:ellipsis;
	}
	#upgradeOperDiv .el-form-item{
		display:inline-block;
		vertical-align:top;
	}
</style>
<div id="upgradeOperDiv">
	<el-form style="width:800px;margin-left:60px;margin-top:50px;" ref="upgradeForm" :model="upgradeForm" :rules="upgradeRules" label-position="top">
		<el-form-item label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='margin-right:140px;' prop="product">
			<el-select v-model="upgradeForm.product">
				<el-option v-for="item in productOptions" :key="item.value" :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="<%=rb.getString("BaoLiuPeiZhi")%>" prop="preserve_setting">
			<div style="width:300px;height:20px;border:1px solid #DEDFE6;padding-top:6px;">
				<el-radio-group v-model="upgradeForm.preserve_setting">
					<el-radio label="1" style="margin-left:20px;margin-right:80px;"><%=rb.getString("Shi")%></el-radio>
					<el-radio label="0"><%=rb.getString("Fou")%></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<div style="position:relative;display:inline-block;margin-right:95px;vertical-align:top;margin-top:20px;">
			<label style="display:block;font-size:14px;margin-bottom:5px"><%=rb.getString("ChuShiBanBen")%></label>
			<div class="queryGroup">
				<el-input v-model="upgradeForm.versionInput" placeholder="<%=rb.getString("QingShuRuHuoXuanZe")%>"></el-input>
				<span @click="selectOriginalVersion" class="el-icon" :class="selectIcon" style="font-size:14px;"></span>
			</div>
			<span @click="addVersion" class="el-icon el-icon-plus" style="vertical-align:middle;margin-left:5px;"></span>
			<el-select ref="selectOriginal" allow-create filterable multiple v-model="upgradeForm.originalVersion" class="oriVersionClass">
				<el-option v-for="item in upgradeForm.originalVersionOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
			</el-select>
			<div style="width:350px;">
				<el-form-item class='suffixItem' v-for='(domain,index) in upgradeForm.versionGroup' style='line-height:16px;'>
					<div class='form-suffix'>
						<span class='text'>{{domain}}</span>
						<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeVersion(domain)'></span>
					</div>
				</el-form-item>
				<p style='color:red;font-size:12px;'>{{errorMsg}}</p>
			</div>
			<el-form-item prop="oriVersion" style="width:300px;">
				<el-input v-model="upgradeForm.oriVersion" v-show=false></el-input>
			</el-form-item>
		</div>
		<el-form-item label="<%=rb.getString("MuBiaoBanBen")%>" style="margin-top:20px;" prop="dest_version">
			<el-select v-model="upgradeForm.dest_version">
				<el-option v-for="item in targetVersionOptions" :key="item.text" :label="item.text" :value="item.text"></el-option>
			</el-select>
		</el-form-item>
	</el-form>
</div>
<script>
	var upgradeVue = new Vue({
		el:"#upgradeOperDiv",
		data(){
			var vm = this;
			var validateItem = function(rule,value,callback){
				if(value == ""){
					callback(new Error("<%=rb.getString("QingXuanZe")%>"))
				}else{
					callback();
				}
			}
			var validateOriVersion = function(rule,value,callback){
				if(vm.upgradeForm.versionGroup.length == 0){
					callback(new Error("<%=rb.getString("ZhiShaoTianJiaYiGe")%>"))
				}else{
					callback();
				}
			}
			return{
				upgradeForm:{
					product:"",
					preserve_setting:"1",
					oriVersion:"",
					dest_version:"",
					originalVersionOptions:[],
					originalVersion:[],
					originalVersionSelect:[],
					versionGroup:[],
					versionInput:"",
				},
				upgradeRules:{
					product:[
						{validator:validateItem}
					],
					dest_version:[
						{validator:validateItem}
					],
					oriVersion:[
						{validator:validateOriVersion}
					]
				},
				productOptions:[],
				targetVersionOptions:[],
				selectIcon:"el-icon-down",
				errorMsg:""
			}
		},
		methods:{
			init(){
				var vm = this;
				axios.post("${ctx}/task/upgrade/getProductType.action",stringify({
                	type: 'all'
                })).then(function(response){
					var data = response.data;
					vm.productOptions = data;
				})
			},
			selectOriginalVersion(){
				this.$refs.selectOriginal.focus();
			},
			addVersion(){
				var vm = this;
				var val = vm.upgradeForm.versionInput;
				if(val == ""){
					vm.errorMsg = "<%=rb.getString("QingShuRu")%>"
				}else if(vm.upgradeForm.versionGroup.length != 0 && vm.upgradeForm.versionGroup.indexOf(val) != -1){
					vm.errorMsg = "<%=rb.getString("YiCunZai")%>"
				}else{
					vm.upgradeForm.originalVersion.push(val);
					vm.errorMsg = "";
					vm.upgradeForm.versionInput = "";
				}
			},
			removeVersion(item){
				var vm = this;
				var index = vm.upgradeForm.versionGroup.indexOf(item);
				if(index !== -1){
					vm.upgradeForm.originalVersion.splice(index,1);
				}
				vm.errorMessage = '';
			},
			submit(){
				var vm = this;
				vm.$refs.upgradeForm.validate(function(valid){
					if(valid){
						var params = {
								product : vm.upgradeForm.product,
								original_version : vm.upgradeForm.versionGroup.toString(),
								dest_version : vm.upgradeForm.dest_version,
								preserve_setting : vm.upgradeForm.preserve_setting
						}
						if(selfVue.operTypeUpgrade == "add"){
							var url = "${ctx}/SON/SelfConfiguration/addSelfUpgradePlans.action"
						}else if(selfVue.operTypeUpgrade == "edit"){
							params.id = selfVue.rowDataUpgrade.id;
							var url = "${ctx}/SON/SelfConfiguration/updateSelfUpgradePlans.action"
						}
						axios.post(url,stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:"<%=rb.getString("ChengGong")%>",
		    						type:'success',
		    					})
                                selfVue.$refs.ctableUpgrade.refresh();
                                selfVue.$refs.slide.hide();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			getInfo(){
				var vm = this;
				vm.upgradeForm.product = selfVue.rowDataUpgrade.product;
				vm.upgradeForm.preserve_setting = selfVue.rowDataUpgrade.preserve_setting;
				vm.upgradeForm.dest_version = selfVue.rowDataUpgrade.dest_version;
				vm.upgradeForm.originalVersion = selfVue.rowDataUpgrade.original_version.split(",")
			}
		},
		watch:{
			"upgradeForm.product":function(val){
				var vm = this;
				axios.post("${ctx}/SON/SelfConfiguration/getOriginalVersionList.action",stringify({product:val})).then(function(response){
					var data = response.data;
					vm.upgradeForm.originalVersionOptions = data;
				})
				//vm.upgradeForm.originalVersion = [];
				var params = {
						product_value : vm.upgradeForm.product
				}
				axios.post("${ctx}/SON/SelfConfiguration/getSelfUpgradeFilePageList.action",stringify(params)).then(function(response){
					var data = response.data;
					vm.targetVersionOptions = data;
				})
			},
			"upgradeForm.originalVersion":function(){
				var vm = this;
				if(vm.upgradeForm.originalVersion.length == 0){
					vm.upgradeForm.versionGroup = [];
				}else{
					vm.upgradeForm.versionGroup = [];
					vm.upgradeForm.originalVersion.map(function(item){
						vm.upgradeForm.versionGroup.push(item);
						vm.errorMessage = '';
					})
				}
			}
		},
		mounted(){
			this.init();
			eventBus.$off("save-upgrade").$on("save-upgrade",this.submit);
			eventBus.$off("edit-upgrade").$on("edit-upgrade",this.getInfo);
		}
	})
</script>