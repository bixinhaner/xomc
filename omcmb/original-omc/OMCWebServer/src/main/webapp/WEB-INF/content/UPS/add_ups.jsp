<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.w300{
		width:350px
	}
	#addUpsContent .dialog-footer{
		text-align: right;
		margin-top: 10px
	}
	.inpBox{
		position: relative;
		height: 50px
	}
	.inpBox span{
		color: #FF0000;
		position: absolute;
		right: 0;
		bottom: 0;
	}
</style>
<div id="addUpsContent">
	<div class="inpBox">
		<%=rb.getString("DianYuanBianMa")%>
		<el-input v-model='input' class="w300"></el-input>
		<span v-if="showError">{{errorMsg}}</span>
	</div>
	<div class='dialog-footer'>
		<el-button type='primary' @click='addUps'><%=rb.getString("TianJia")%></el-button>
		<el-button @click='closeWinAddUps'><%=rb.getString("QuXiao")%></el-button>
	</div>
</div>
<script>
	var addDeviceVue = new Vue({
		el:'#addUpsContent',
		data(){
			return{
				input:'',
				errorMsg:'',
				showError:false,
				gsId:''
			}
		},
		methods:{
			init(type,id,enbFlag){
				this.gsId = id
			},
			addUps(){ //添加 UPS
				var vm = this,
					value = vm.input,
					obj = {},
					url = '${ctx}/ups/addDevice.action',
					reg = /^[0-9a-zA-Z]*$/;	
				
				// 在点击确定之前先验证
				if(value == ''){
					vm.showError = true,
					vm.errorMsg = "<%=rb.getString("UPSSNBuNengWeiKong")%>";
					return	
				}else{
					if(value.length === 19 && reg.test(value)){
						vm.showError = false
					}else{
						vm.showError = true,
						vm.errorMsg = "<%=rb.getString("QingShuRuZhengQueUPSSn")%>";
						return
					}
				}
				obj.group_id = vm.gsId,
				obj.serialNumber = value;
				axios.post(url,stringify(obj)).then(function(response){
					let data = response.data;
					if(data.success){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});	
						vm.closeWinAddUps()
					}else{
						vm.$message({
							type:'error',
							message:data.message,
						})
					}
				})
			},
			closeWinAddUps(){ // 关闭UPS 弹窗
				eventBus.$emit('close-dialog');
			}
		},
		mounted(){
			eventBus.$off('open-dialog').$on('open-dialog',this.init)
			
		},
	})
</script>