<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#ipsecDiv .el-input{
		width:300px;
	}
	#ipsecDiv .el-form-item{
		margin-right: 80px;
		display: inline-block;
		vertical-align: top;
	}
</style>
<div id="ipsecDiv">
	<div class="group-title not-extend" style='margin-top: 20px; margin-left: 40px;'>
		<span class="title-icon"></span>
		<span class="title-text">Ipsec Tunnel</span>
	</div>
	<el-form ref="ipsecForm" :disabled="viewDisabled" :model="ipsecForm" :rules="ipsecRules" label-position="top" style="margin-left:45px;margin-top:20px;">
		<el-form-item v-for="item in paramList" :label="item.name" :prop="item.MIB_DN">
			<el-input v-if="item.showInput" v-model="ipsecForm[item.MIB_DN]"></el-input>
			<el-select v-if="item.showSelect" v-model="ipsecForm[item.MIB_DN]">
				<el-option v-for="item1 in item.result" :label="item1.title" :value="item1.value">
				
				</el-option>
			</el-select>
		</el-form-item>
	</el-form>
</div>
<script>
	var validateItem;
	var ipsecVue = new Vue({
		el:"#ipsecDiv",
		data(){
			var vm = this;
			return{
				paramList:[],
				ipsecForm:{
					
				},
				ipsecRules:{
					
				},
				viewDisabled:false
			}
		},
		methods:{
			init(){
				var vm = this;
				(enbAddOrEditConfigVue.paramList || []).map(function(item){
					
					var type = item.V_TYPE.substring(0,item.V_TYPE.indexOf("-"));
					var MIB_DN = item.MIB_DN;
					var TITLE = item.TITLE;
					var JS_REGEX = item.JS_REGEX;
					if(type == "string" || type == "stringList"){
						var minVal = item.V_TYPE.substring(item.V_TYPE.indexOf("[")+1,item.V_TYPE.indexOf(":"));
						var maxVal = item.V_TYPE.substring(item.V_TYPE.indexOf(":")+1,item.V_TYPE.indexOf("]"));
						var obj = {
								name : item.PARAM_NAME,
								showInput:true,
								showSelect:false,
								MIB_DN:MIB_DN
						}
						var validateItem = function(rule,value,callback){
							if(MIB_DN == "KEYLIFE"){
								var reg = eval("(" + JS_REGEX + ")")
								if(value == "" || value == null){
									callback();
								}
								if(reg.test(value) && value.length <= maxVal){
									if(vm.ipsecForm.REKEYMARGIN.includes("h")){
										var rekeymargin = vm.ipsecForm.REKEYMARGIN.substring(0,vm.ipsecForm.REKEYMARGIN.length-1)*60;
									}else if(vm.ipsecForm.REKEYMARGIN.includes("m")){
										var rekeymargin = vm.ipsecForm.REKEYMARGIN.substring(0,vm.ipsecForm.REKEYMARGIN.length-1);
									}else if(vm.ipsecForm.REKEYMARGIN.includes("d")){
										var rekeymargin = vm.ipsecForm.REKEYMARGIN.substring(0,vm.ipsecForm.REKEYMARGIN.length-1)*24*60;
									}
									if(value.includes("h")){
										value = value.substring(0,value.length-1)*60;
										if(value - rekeymargin*3 >= 0){
											callback();
										}else{
											callback(new Error("Keylife>=Rekeymargin*3"))
										}
									}else if(value.includes("m")){
										value = value.substring(0,value.length-1);
										if(value - rekeymargin*3 >= 0){
											callback();
										}else{
											callback(new Error("Keylife>=Rekeymargin*3"));
										}
									}else if(value.includes("d")){
										value = (value.substring(0,value.length-1))*24*60;
										if(value - rekeymargin*3 >= 0){
											callback();
										}else{
											callback(new Error("Keylife>=Rekeymargin*3"));
										}
									}else{
										callback();
									}
								}else{
									callback(new Error("number+h/m/d"))
								}
								vm.$refs.ipsecForm.validateField("IKELIFETIME");
							}else if(MIB_DN == "IKELIFETIME"){
								var reg = eval("(" + JS_REGEX + ")")
								if(value == "" || value == null){
									callback();
								}else{
									if(reg.test(value) && value.length <= maxVal){
										
										if(vm.ipsecForm.KEYLIFE.includes("h")){
											var margin = vm.ipsecForm.KEYLIFE.substring(0,vm.ipsecForm.KEYLIFE.length-1)*60;
										}else if(vm.ipsecForm.KEYLIFE.includes("m")){
											var margin = vm.ipsecForm.KEYLIFE.substring(0,vm.ipsecForm.KEYLIFE.length-1);
										}else if(vm.ipsecForm.KEYLIFE.includes("d")){
											var margin = vm.ipsecForm.KEYLIFE.substring(0,vm.ipsecForm.KEYLIFE.length-1)*24*60;
										}else if(vm.ipsecForm.KEYLIFE == ""){
											var margin = 0;
										}
										if(value.includes("h")){
											value = value.substring(0,value.length-1)*60;
											if(value - margin >= 0){
												callback();
											}else{
												callback(new Error("Ikelifetime>=Keylife"));
											}
										}else if(value.includes("m")){
											value = (value.substring(0,value.length-1));
											if(value - margin >= 0){
												callback();
											}else{
												callback(new Error("Ikelifetime>=Keylife"));
											}
										}else if(value.includes("d")){
											value = (value.substring(0,value.length-1)*24*60);
											if(value - margin >= 0){
												callback();
											}else{
												callback(new Error("Ikelifetime>=Keylife"));
											}
										}else{
											callback();
										}
									}else{
										callback(new Error("number+h/m/d"));
									}
								}
							}else if(MIB_DN =="REKEYMARGIN"){
								var reg = eval("(" + JS_REGEX + ")")
								if(value == "" || value == null){
									callback();
								}else{
									if(reg.test(value) && value.length <= maxVal){
										if(value.includes("h")){
											value = (value.substring(0,value.length-1))*60;
										}else if(value.includes("m")){
											value = value.substring(0,value.length-1);
										}else if(value.includes("d")){
											value = (value.substring(0,value.length-1))*24*60;
										}
										if(value >= 5){
											callback();
										}else{
											callback(new Error("Rekeymargin>=5min"));
										}
									}else{
										callback(new Error("number+h/m/d"));
									}
								}
								vm.$refs.ipsecForm.validateField("KEYLIFE");
							}else{
								if(value == "" || value == null){
									if(MIB_DN == "TUNNEL_NAME" || MIB_DN == "TUNNEL_GATEWAY" ){
										callback(new Error(TITLE));
									}else{
										callback();
									}
								}else{
									var reg;
									if(JS_REGEX == "no_zh"){
										reg = /^(?:(?![\u4E00-\u9FA5]|[\uFE30-\uFFA0]).)+$/;
									}else{
										reg = eval("(" + JS_REGEX + ")");
									}
									if(reg.test(value) && value.length <= maxVal){
										callback();
									}else{
										callback(new Error(TITLE));
									}
								}
							}
						}
						var rule = [
							{validator:validateItem,trigger:'blur'}
						]
						vm.ipsecRules[MIB_DN] = rule;
						vm.$set(vm.ipsecForm,MIB_DN,"");
						
						vm.paramList.push(obj)
					}
					if(type == "enum"){
						var index = item.V_TYPE.substring(item.V_TYPE.indexOf("{"),item.V_TYPE.lastIndexOf("}")+1);
						var title = index.substring(0,index.indexOf("}")+1);
						title = title.substring(title.indexOf("{")+1,title.indexOf("}")).split(",");
						var value = index.substring(index.lastIndexOf("{"))
						value = value.substring(value.indexOf("{")+1,value.indexOf("}")).split(",");
						let result = [];
						title.map(function(item,index){
							result.push({title:item,value:value[index]})
						})
						var obj = {
								name : item.PARAM_NAME,
								showInput:false,
								showSelect:true,
								result:result,
								MIB_DN:MIB_DN
						}
						vm.$set(vm.ipsecForm,MIB_DN,value[0]);
						vm.paramList.push(obj)
					}
				})
				if(enbAddOrEditConfigVue.operTypeIpsec == "view"){
					vm.viewDisabled = true;
				}
			},
			saveIpsec(){
				var vm = this;
				var params = {};
				vm.$refs.ipsecForm.validate(function(valid){
					if(valid){
						if(enbAddOrEditConfigVue.operTypeIpsec == "add"){
							params.IPSEC_INDEX = enbAddOrEditConfigVue.settingForm.ipsecList.length + 1;
							params.PLAN_ID = enbAddOrEditConfigVue.rowDataPlan.serial_number;
							for(var key in vm.ipsecForm){
								params[key] = vm.ipsecForm[key]
							}
							enbAddOrEditConfigVue.settingForm.ipsecList.push(params);
							enbAddOrEditConfigVue.$refs.slide.hide();
							enbAddOrEditConfigVue.$refs.settingForm.validateField("ipsecLength");
						}
						if(enbAddOrEditConfigVue.operTypeIpsec == "edit"){
							params.IPSEC_INDEX = enbAddOrEditConfigVue.rowDataIpsec.IPSEC_INDEX;
							params.PLAN_ID = enbAddOrEditConfigVue.rowDataPlan.serial_number;
							for(var key in vm.ipsecForm){
								params[key] = vm.ipsecForm[key]
							}
							var ipsecArr = enbAddOrEditConfigVue.settingForm.ipsecList.map(function(item){
								return item.IPSEC_INDEX;
							})
							var index = ipsecArr.indexOf(params.IPSEC_INDEX);
							enbAddOrEditConfigVue.settingForm.ipsecList.splice(index,1,params);
							enbAddOrEditConfigVue.$refs.slide.hide();
						}
					}
				})
			},
			getInfo(){
				var vm = this;
				for(var key in vm.ipsecForm){
					vm.ipsecForm[key] = enbAddOrEditConfigVue.rowDataIpsec[key]
				}
			}
		},
		mounted(){
			this.init();
			eventBus.$off("edit-ipsec").$on("edit-ipsec",this.getInfo)
			eventBus.$off("save-ipsec").$on("save-ipsec",this.saveIpsec)
		}
	})
</script>